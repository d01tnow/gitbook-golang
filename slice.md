# Slice

切片是 Go 中的一种基本数据结构. 切片本身是一个只读对象.

## 底层实现

``` go
type slice struct {
  array unsafe.Pointer
  len int
  cap int
}
```

所以 unsafe.Sizeof([]T{}) == 3 words(3个字长, 1 word 在 32bit操作系统中代表 4 个字节, 在 64bit 操作系统中代表 8 个字节).
可以用在 slice 上的内建函数有 len, cap, make, copy, append 等.
不管 slice 是否可寻址, slice 索引操作是可寻址的( sl[x] is addressable. &sl[x] ).
