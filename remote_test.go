package iomonkey

import (
	"testing"
	"github.com/ncw/go-acd"
	"strings"
	"fmt"
)


func TestCannotGetNonExistantFolder(t *testing.T) {
	remote, err := NewAcdRemote()
	if err != nil {
		t.Fatal(err);
	}

	client,err :=  remote.CreateClient()
	if err != nil {
		t.Fatal(err);
	}

	if _,err := client.getNode("nonExistant"); err != acd.ErrorNodeNotFound {
		t.Fatal(err);
	}
}

func TestCanCreateAndRemoveFolder(t *testing.T) {
	remote, err := NewAcdRemote()
	if err != nil {
		t.Fatal(err);
	}

	client,err :=  remote.CreateClient()
	if err != nil {
		t.Fatal(err);
	}

	root,err  := client.getRoot()
	if err != nil {
		t.Fatal(err);
	}

	testFolder,_,err := root.Typed().(*acd.Folder).CreateFolder("test")
	if err != nil {
		t.Fatal(err);
	}

	_,err = testFolder.Trash()
	if err != nil {
		t.Fatal(err);
	}

	if _,err := client.getNode("test"); err != acd.ErrorNodeNotFound {
		t.Fatal(err);
	}
}

func TestGetOrCreateFolder(t *testing.T) {
	remote, err := NewAcdRemote()
	if err != nil {
		t.Fatal(err);
	}

	client,err :=  remote.CreateClient()
	if err != nil {
		t.Fatal(err);
	}

	previousNode,err := client.getRoot()
	if err != nil {
		t.Fatal(err)
	}
	folder := "test/a/b/c"
	parts := client.split(folder)
	for i,_ := range parts{
		p := strings.Join(parts[0:i+1],"/");
		fmt.Println("Getting node : "+p)
		node,err := client.getNode(p)
		if err != nil {
			t.Fatal(err)
		}

		if node == nil {
			newFolder,_,err := previousNode.Typed().(*acd.Folder).CreateFolder(parts[i])
			if err != nil {
				t.Fatal(err)
			}
			node = newFolder.Node
		}
		previousNode = node
	}


}