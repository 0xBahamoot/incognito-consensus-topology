package icnwtopology

import (
	"sync"
)

type Tree struct {
	root      string
	leftTree  *AVLTree
	rightTree *AVLTree
}

func compareString(a interface{}, b interface{}) bool {
	if a.(string) < b.(string) {
		return false
	}
	return true
}

func buildTree(list []string) *AVLTree {
	tree := &AVLTree{}
	for _, i := range list {
		tree.Insert(i, "", compareString)
	}
	return tree
}

func CreateTree(root string, peerList []string) (*Tree, error) {
	var result Tree

	peerListWithoutRoot := make([]string, len(peerList))
	copy(peerListWithoutRoot, peerList)

	for idx, peerID := range peerListWithoutRoot {
		if peerID == root {
			copy(peerListWithoutRoot[idx:], peerListWithoutRoot[idx+1:])
			peerListWithoutRoot[len(peerListWithoutRoot)-1] = ""
			peerListWithoutRoot = peerListWithoutRoot[:len(peerListWithoutRoot)-1]
		}
	}

	// randomizeList := func() {
	// 	h := md5.New()
	// 	io.WriteString(h, root)
	// 	var seed uint64 = binary.BigEndian.Uint64(h.Sum(nil))
	// 	rand.Seed(int64(seed))
	// 	rand.Shuffle(len(peerListWithoutRoot), func(i, j int) {
	// 		peerListWithoutRoot[i], peerListWithoutRoot[j] = peerListWithoutRoot[j], peerListWithoutRoot[i]
	// 	})
	// }
	// randomizeList()

	maxLeftTreeMember := len(peerList) / 2
	leftTreeMember := make([]string, len(peerListWithoutRoot[:maxLeftTreeMember]))
	rightTreeMember := make([]string, len(peerListWithoutRoot[maxLeftTreeMember:]))

	copy(leftTreeMember, peerListWithoutRoot[:maxLeftTreeMember])
	copy(rightTreeMember, peerListWithoutRoot[maxLeftTreeMember:])

	var wg sync.WaitGroup
	wg.Add(2)
	result.root = root
	go func() {
		result.leftTree = buildTree(leftTreeMember)
		wg.Done()
	}()
	go func() {
		result.rightTree = buildTree(rightTreeMember)
		wg.Done()
	}()
	wg.Wait()

	return &result, nil
}

func (tr *Tree) GetRoot() (root string) {
	return tr.root
}

func (tr *Tree) GetParent(key string) (string, error) {
	if node := tr.leftTree.root.searchRootOfNode(key); node != nil {
		return node.key.(string), nil
	}
	if node := tr.rightTree.root.searchRootOfNode(key); node != nil {
		return node.key.(string), nil
	}
	return "", ErrKeyNotFound
}

func (tr *Tree) GetChild(key string) (left, right string, err error) {
	if key == tr.root {
		if tr.leftTree != nil {
			left = tr.leftTree.root.key.(string)
		}
		if tr.rightTree != nil {
			right = tr.rightTree.root.key.(string)
		}
		return
	}

	if node := tr.leftTree.root.search(key); node != nil {
		if node.left != nil {
			left = node.left.key.(string)
		}
		if node.right != nil {
			right = node.right.key.(string)
		}
	} else {
		if node := tr.rightTree.root.search(key); node != nil {
			if node.left != nil {
				left = node.left.key.(string)
			}
			if node.right != nil {
				right = node.right.key.(string)
			}
		} else {
			err = ErrKeyNotFound
		}
	}

	return
}

func (tr *Tree) GetNodeHeight(key string) int64 {
	if key == tr.root {
		return tr.leftTree.root.height + 1
	}
	if node := tr.leftTree.root.search(key); node != nil {
		return node.height
	} else {
		if node := tr.rightTree.root.search(key); node != nil {
			return node.height
		}
	}
	return 0
}
