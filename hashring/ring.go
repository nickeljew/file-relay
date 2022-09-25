package hashring

import (
	//"crypto/md5"
	//"math"
)


const (
	PartitionCount = 4294967296 //int(math.Pow(2, 32))
)


type Node struct {
	name string
}

type VNode struct {
	prev *VNode
	next *VNode
	//
}

type Ring struct {

}
