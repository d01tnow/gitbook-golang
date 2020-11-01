package list

type node struct {
	value int   // 数据域
	next  *node // 后续节点
}

// LinkedList 单链表
type LinkedList struct {
	head *node
}

// NewLinkedList 创建新链表
func NewLinkedList() *LinkedList {
	// 带头节点
	return &LinkedList{}
}
