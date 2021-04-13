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

## 切片

### 子切片

```go
// 以下 low, high, max 满足条件: 0 <= low <= high <= max <= cap(baseContainer)
// 取子切片时 low 可以超过 len(baseContainer) , 但必须满足上面的条件.
subSlice := baseContainer[low : high] // 双下标形式
subSlice := baseContainer[low : high : max] // 三下标形式
// 双下标形式等同于以下表述的三下标形式
subSlice := baseContainer[low : high : cap(baseContainer)]
// subSlice 的长度和容量
l := len(subSlice)  // l = high - low
c := cap(subSlice)  // c = max - low
```

