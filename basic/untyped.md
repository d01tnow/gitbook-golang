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

  