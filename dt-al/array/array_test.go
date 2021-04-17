package array

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDefine(t *testing.T) {
	Convey("测试数组定义", t, func() {

		Convey("var a [5]int", func() {
			var a [5]int
			So(len(a), ShouldEqual, 5)
			for i := 0; i < len(a); i++ {
				So(a[i], ShouldEqual, 0)
			}
		})
		Convey("b := [5]int{1,2,3,}", func() {
			b := [5]int{0, 1, 2} // [5]int{0,1,2,0,0}
			So(len(b), ShouldEqual, 5)
			for i := 0; i < 3; i++ {
				So(b[i], ShouldEqual, i)
			}
			So(b[3], ShouldEqual, 0)
			So(b[4], ShouldEqual, 0)
		})
		Convey("var twoD [2][3]int", func() {
			var twoD [2][3]int
			// 上面定义的 2 维数组实际上是类型为 [3]int 的 1 维数组
			So(len(twoD), ShouldEqual, 2)
			// 赋值
			for i := 0; i < 2; i++ {
				for j := 0; j < 3; j++ {
					twoD[i][j] = i + j
					So(twoD[i][j], ShouldEqual, i+j)
				}
			}
		})

	})
}
