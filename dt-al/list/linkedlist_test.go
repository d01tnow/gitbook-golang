package list

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNew(t *testing.T) {
	Convey("测试 NewLinkedList()", t, func() {
		a := NewLinkedList()
		So(a, ShouldNotBeNil)
	})
}
