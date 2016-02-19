package iomonkey
import (
	"github.com/ncw/go-acd"
	"io/ioutil"
	"path/filepath"
	"encoding/json"
"net/http"
	"sync"
	"time"
	"log"
	"strings"

)

const REMOTE_PREFIX = "iomonkey"
const CACHE_FILE = "/tmp/iomonkey.cache"

type Remote interface {
	CreateClient() (Client,error)
}

type Client interface {
	Upload(local,remote string) error
}

type AcdRemote struct {
	oauthClient *http.Client

}

type AcdClient struct {
	rwMutex sync.RWMutex
	nodes map[string]*acd.Node
	client *acd.Client
	notifyWrite chan bool

}

func NewAcdRemote() (*AcdRemote,error) {
	//log.Println("Authorizing with remote")
	oauthClient, err := Authorize()
	if err != nil {
		return nil,err
	}
	return &AcdRemote{
		oauthClient: oauthClient,
	},nil
}

func (a *AcdRemote) CreateClient() (*AcdClient, error) {
	apiClient := acd.NewClient(a.oauthClient)

	client := &AcdClient{
		nodes: make(map[string]*acd.Node),
		client:apiClient,
		notifyWrite:make(chan bool,1),
	}

	go client.cacheWriter()
	client.loadCache()

	return  client,nil
}

func (a *AcdClient) Upload(local,remote string) error {
	log.Printf("INFO: Uploading %s to remote %s\n", local,remote)
	dir,_ := filepath.Split(remote)
	dir = filepath.Join(REMOTE_PREFIX,dir)

	//a.getOrCreateFolder(dir)

	log.Println("getOrCreateFolder "+dir)
	//fmt.Println(file)
	//hierarchy := filepath.SplitList(dir)
	return nil
}

func (a *AcdClient) isRoot(node *acd.Node) bool {
	return len(node.Parents) == 0
}

func (a *AcdClient) createProperNode(node *acd.Node) *acd.Node{
	properNode := acd.NodeFromId(*node.Id,a.client.Nodes)
	properNode.ContentProperties = node.ContentProperties
	properNode.Kind = node.Kind
	properNode.ModifiedDate = node.ModifiedDate
	properNode.Name = node.Name
	properNode.Parents = node.Parents
	properNode.Status = node.Status
	properNode.TempURL = node.TempURL
	return properNode
}


func (a *AcdClient) cacheNode(node *acd.Node){
	log.Printf("DEBUG: Caching node")
	parent := node
	name := ""
	if node.Name != nil {
		name = *node.Name
	}
	key := filepath.Join(name)
	for {
		parent := a.getFirstParent(parent)
		if parent == nil {
			break
		}
		if !a.isRoot(parent){
			key = filepath.Join(*parent.Name,key)
		}
	}

	a.rwMutex.Lock()
	a.nodes[key] = node
	a.rwMutex.Unlock()
	log.Println("INFO: sending write notification")
	a.notifyWrite<-true
}

func (a *AcdClient) getFirstNode(path string) (*acd.Node, string, error){
	log.Printf("DEBUG: getFirstNode %s\n",path)
	parts := a.split(path)
	for i := len(parts); i > 0 ; i-- {
		p := strings.Join(parts[0:i],"/")
		node := a.getCachedNode(p)
		if node != nil{
			log.Printf("INFO: Found cached node @ %s\n",p)
			return node,p,nil
		}
	}
	log.Println("DEBUG: Only root left")
	root,err := a.getRoot()
	return root,"",err
}


func (a *AcdClient) getCachedNode(path string) (node *acd.Node){
	a.rwMutex.RLock()
	node = a.nodes[path]
	a.rwMutex.RUnlock()
	if node != nil {
		log.Printf("INFO: getting node from cache @ %s\n",path)
	}
	return
}

func (a *AcdClient) getNode(path string) (node *acd.Node, err error){
	node = a.getCachedNode(path);
	if node == nil {
		node,err = a.getRemoteNode(path);
	}

	return
}

func (a *AcdClient) getRemoteNode(path string) (*acd.Node, error){
	node,firstPath,err := a.getFirstNode(path)
	if err != nil {
		return nil,err;
	}
	log.Printf("Asking remote for %s\n",path)
	path = strings.TrimPrefix(path,firstPath)
	if path != "" {
		for _,p :=  range a.split(path){
			node,_,err = node.Typed().(*acd.Folder).GetNode(p)
			if err == acd.ErrorNodeNotFound {
				return nil,nil
			}else if err != nil {
				return nil,err
			}
			a.cacheNode(node)
		}
	}
	return node,nil
}


func(a *AcdClient) split(path string) []string{
	return strings.Split(strings.Trim(path,"/"),"/");
}

func (a *AcdClient) getParent(path string) (*acd.Node, error){
	if !strings.Contains(path,"/"){
		return a.getRoot()
	}
	parts := a.split(path)
	return a.getNode(strings.Join(parts[0:len(parts)-1],"/"))
}

func (a *AcdClient) getRoot() (root *acd.Node,err error){
	root,err = a.getNode("")

	if root == nil {
		var folder *acd.Folder
		folder,_,err = a.client.Nodes.GetRoot()
		root = folder.Node
		a.cacheNode(root)
	}
	return
}

func (a *AcdClient) getFirstParent(node *acd.Node) *acd.Node{
	if a.isRoot(node){
		return nil
	}

	a.rwMutex.RLock()
	defer a.rwMutex.RLock()

	parentId := node.Parents[0]
	for  _,n := range  a.nodes {
		if *n.Id == parentId {
			return n
		}
	}

	return nil
}

func(a *AcdClient) loadCache(){
	b,err := ioutil.ReadFile(CACHE_FILE)
	if err != nil {
		log.Printf("WARN: could not read acd client cache, err: %s\n",err)
		return
	}
	nodes := make(map[string]*acd.Node)
	json.Unmarshal(b,&nodes)

	for _,node := range nodes {
		node = a.createProperNode(node)
		a.cacheNode(node)
	}
}

func (a *AcdClient) cacheWriter(){
	var newNodes bool
	for {
		select {
			case <-a.notifyWrite:
				log.Println("INFO: received write notification")
				newNodes = true
			case <-time.After(time.Second*5):
				if !newNodes {
					break
				}
				log.Println("INFO: Writing node cache")
				a.rwMutex.RLock()
				b, _ := json.Marshal(a.nodes)
				ioutil.WriteFile(CACHE_FILE, b, 0755)
				newNodes = false
				a.rwMutex.RUnlock()
		}
	}
}

