# 类型系统



## Undefined type And Defined type

理解 defined type, undefined type, underlying type 是理解类型转换的基础.

### 概念

* 未定义类型和已定义类型. 所有的基本类型都是定义类型。一个非定义类型一定是一个组合类型。

* 底层类型(underlying type): 

  * 内置基本类型的底层类型是它自己.
  * 组合类型(未定义类型)的底层类型是它自己.
  * unsafe.Pointer 的底层类型是它自己.
  * 新定义的类型和它的源类型共享底层类型.

* 类型追溯方法: 追溯到内置基本类型或未定义类型结束.

  ```go
  // 这2个类型的底层类型均为内置类型int。
  type (
  	MyInt int // int 是已定义类型, MyInt 也是
  	Age   MyInt // MyInt 和 Age 都是已定义类型
  )
  
  // 下面这三个新声明的类型的底层类型各不相同。
  type (
  	IntSlice   []int   // 底层类型为[]int , []int 是未定义类型, IntSlice 是已定义类型
  	MyIntSlice []MyInt // 底层类型为[]MyInt
  	AgeSlice   []Age   // 底层类型为[]Age
  )
  
  // 类型[]Age、Ages和AgeSlice的底层类型均为[]Age。
  type Ages AgeSlice
  
  // 类型追溯
  // MyInt -> int
  // Age -> MyInt -> int
  // IntSlice -> []int
  // MyIntSlice -> []MyInt // 组合类型结束
  // AgeSlice -> []Age 
  ```

  golang 已经定义的基本类型, 都是 defined type:

* 字符串类型: string
* 布尔类型: bool
* 数值类型:
  * int8, int16, int32 (rune), int64, uint8 (byte), uint16, uint32, uint64, int, uint, uintptr
  * float32, float64
  * complex64, complex128

组合类型( 字面量都是 undefined type):

* 指针: *int
* 数组: [5]int
* 切片: []int
* map: map[string]int
* 函数: func (int) error 
* channel: chan []int
* 结构体: struct { x int "foo"}
* 接口: interface { f([]int) int }

别名( type T = X ): 要看 X 是已定义类型还是未定义类型.

```go
// defined type: int, float32 ...
// 重新定义了新类型 MyInt. 它和 int 不是同一类型. 但值可以显式转换.
type MyInt int
// StrSlice 是已定义类型, []string 组合类型是未定义类型.
type StrSlice []string

// defined type
type AliasInt = int

// undefined type
type AliasStrSlice = []string

```

### 类型转换

显式转换: y = (T)(x). T 代表类型, x 代表值或变量, y 代表变量. 当 T 是类型名时可以简写为 T(x). 

隐式转换:  y = x

首先要清楚:

* 重新定义的类型与其源类型**不是**同一类型.
* 重新定义的类型的值与其源类型的值**可以显式**转换.
* 如果两个类型就是同一类型(比如: 通过别名方式声明的两个类型), 那么用它们声明的变量的值可以**隐式**转换. (uint8 <-> type, int32 <-> rune, []uint8 <-> []byte).
* 2 个结构体中(类型相同但字段名不同), 或者(字段名,类型名都相同, 但是顺序不同), 这两个结构体不是同一类型.

#### 底层类型转换

* 如果 2 个变量的底层类型相同(忽略掉结构体字段的标签), 那么, 这两个变量可以显式转换.
* 如果 Tx 和 Ty 至少有一个是未定义类型且共享底层类型(考虑结构体标签), 那么, 这两个类型的变量可以隐式转换.
* 如果类型`Tx`和`T`的底层类型不同，但是两者都是非定义的指针类型并且它们的基类型的底层类型相同（忽略掉结构体字段标签），则`x`可以（而且只能）被显式转换为类型`T`

```go
type IntSlice []int
type MyIntSlice IntSlice // 底层类型 -> IntSlice -> []int
var x struct {
  a int "foo"
  b string
} // x 的类型是未定义类型.
var y struct {
  a int "bar"
  b string
}
var i IntSlice
var n MyIntSlice
// i 和 n 的类型都是已定义类型, 只能通过显示转换
i = IntSlice(n)
x = struct {
  a int "foo"
  b string
}(y) // 未定义类型的显示转换不能忽略标签.
// 底层类型不同的指针
type MyIntPtr *int
var pi *int
var myPi MyIntPtr
// 不能隐式转换 pi = myPi
// 可以显示转换
pi = (*int)(myPi)


```



# Untyped value

Untyped value(类型不确定值) 和 Type value(类型确定值).

* 类型不确定值可以显示或隐式转换为类型确定值. 类型确定的变量只能显式转换.
* untyped const value 隐式转换, **不允许** 溢出(编译错误). 隐式转换为 var 变量的值, 可以溢出, 溢出后被 truncate.

字面量是 Untyped value. 对于大多数类型不确定值来说，它们各自都有一个默认类型， 除了预声明的`nil`。`nil`是没有默认类型的。

一个字面（常）量的默认类型取决于它的它为何种字面量形式：

- 一个字符串字面量的默认类型是预声明的`string`类型。

- 一个布尔字面量的默认类型是预声明的`bool`类型。

- 一个整数型字面量的默认类型是预声明的`int`类型。

- 一个rune字面量的默认类型是预声明的`rune`（亦即`int32`）类型。

- 一个浮点数字面量的默认类型是预声明的`float64`类型。

- 如果一个字面量含有虚部字面量，则此字面量的默认类型是预声明的`complex128`类型。

  ```go
  # untyped value 溢出
  package main
  import "fmt"
  func constOverflow() {
    const a int8 = -1 
    
    // 编译错误. -128 是 untyped const, a 是 const int8. const 表达式编译器求值. 计算结果(128)还是 const, 不允许溢出, 编译错误.
    #var b int8 = -128 / a 
   	fmt.Printf("0b%b\n", a)
  
  }
  func varOverflow() {
    var a int8 = -1 
    // -128 是 untyped const, a 是 int8. 表达式式的结果(128, 0b10000000)是 var int8, 允许溢出, 最高位作为符号位, 刚好是 -128 的补码.
    var b int8 = -128 / a
    fmt.Print(b)
  }
  func main() {
    constOverflow()
    varOverflow()
  }
  ```

  