package main

import (
	"github.com/netbrain/iomonkey"
	"log"
	"os"

	"os/signal"
	"syscall"
	"fmt"
)

func init() {
	log.SetFlags(log.Llongfile)
}


func main() {
	remote,err := iomonkey.NewAcdRemote()
	if err != nil {
		log.Fatal(err)
	}

	return

	client, err := remote.CreateClient()
	if err != nil {
		log.Fatal(err)
	}

	client.Upload("/tmp/t","Videos/MISSING_EXIF/IMG_0034.MOV")
	//Videos/MISSING_EXIF/IMG_0034.MOV
	return


	automounter, err := iomonkey.NewAutoMounter()
	if err != nil {
		log.Fatal(err)
	}
	go onAutomount(automounter)


	//Wait for term or interrupt signal
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	//Cleanup
	automounter.Close()
}

func onAutomount(automounter *iomonkey.AutoMounter){
	for mountEvent := range automounter.Listen() {
		if mountEvent.Error != nil {
			log.Println(mountEvent.Error)
			continue
		}

		go scanFiles(mountEvent)
	}
}


func scanFiles(mountEvent *iomonkey.MountEvent){
	log.Println("Scanning files, this may take some time...")
	fileScanner := iomonkey.NewFileScanner(mountEvent.Mount.Target)
	fileChan,total := fileScanner.Files()
	log.Printf("Found a total of %d files\n",total)
	for f := range fileChan {
		log.Printf("Processing %s\n",f.LocalPath)
		fmt.Println(f.RemotePath)
	}
}
/*if os.Geteuid() != 0 {
		log.Fatal("Not root!")
	}

	log.Println("Authorizing with remote")
	oauthClient, err := iomonkey.Authorize()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Creating client")
	client := acd.NewClient(oauthClient)

	log.Println("Getting root directory")
	root,_,err := client.Nodes.GetRoot()
	if err != nil {
		log.Fatal(err)
	}

	root,err = getOrCreateFolder(root,REMOTE_PREFIX)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening for device to plug&play")
	for event := range iomonkey.Listen(){
		log.Println(event.Devpath)
		go func(){
			devName := event.Vars["DEVNAME"]
			src := "/dev/" + devName
			target := "/mnt/" + devName

			if err := iomonkey.Mount(src,target); err != nil {
				log.Println(err);
				return
			}
			defer func(){
				log.Printf("Unmounting '%s'\n",target)
				if err := iomonkey.Unmount(target); err != nil {
					log.Println("Failed to unmount '%s', err: %s",target)
				}
			}()

			for fileMapping := range iomonkey.WalkFiles(target) {
				log.Printf("Uploading '%s' to '%s/%s'\n",fileMapping.LocalPath,REMOTE_PREFIX,fileMapping.RemotePath)
				currentRoot,err := getOrCreateFolder(root,filepath.Dir(fileMapping.RemotePath))
				if err != nil {
					log.Println(err)
				}
				_,_,err = currentRoot.Upload(fileMapping.LocalPath,filepath.Base(fileMapping.RemotePath))
				if err != nil {
					log.Println(err)
				}
			}
		}()
	}

}

func getOrCreateFolder(root *acd.Folder,folder string) (*acd.Folder,error) {
	currentRoot := root
	for _,folder := range strings.Split(folder,"/"){
		if folder != "" {
			log.Println("Getting subdirectory: " + folder)
			node, _, err := currentRoot.GetFolder(folder)
			if err == acd.ErrorNodeNotFound {
				log.Println("Subderectory doesn't exist, creating it...")
				node,_,err = currentRoot.CreateFolder(folder)
				if err != nil {
					return nil,err
				}
			}else if err != nil {
				return nil,err
			}
			currentRoot = node
		}
	}
	return currentRoot,nil
}
*/
