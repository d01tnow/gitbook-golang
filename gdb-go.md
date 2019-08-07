# GDB 调试 GO

## 加载特性

为了能启用 info goroutines 特性, 需要按如下方法编辑 ~/.gdbinit . 如果没加, 启动 gdb 时会用 warning 提示的.

``` shell
# /usr/lib/go 为 golang 的安装路径
echo "add-auto-load-safe-path /usr/lib/go/src/runtime/runtime-gdb.py" > ~/.gdbinit
```

## 编译选项

为了更好的调试 golang 程序, 需要传递如下参数:

1. -ldflags "-s" : 忽略 debug 的打印信息.
2. -gcflags "-N -l" : 让编译器不做优化.

``` shell
# 例子
go build -gcflags "-N -l" main.go
```

## 启动

``` shell
gdb executable_file
# 定位 coredump 产生原因
gdb executable_file [-c] coredump_file
# 调试运行中的程序
gdb -p pid
```

## 常用命令

命令支持 tab 补全.
GDB 按 \<RET\> -- 回车, 默认执行上一条命令.

- 启动调试: gdb \<app\>
- list, l [行数]: 显示源代码. 默认行数为 10.
- break, b \<[文件名:]行号 [if 表达式]\>/\<函数名\>: 在源码的指定行 或 指定函数处(需要加 package 名, 比如: main.main)设置断点. 条件表达式 -- 满足条件后才暂停运行. 比如: b 10  if v1==v2. 即: 当变量 v1 == v2 时才暂停运行.
- delete, d 断点序号: 删除断点. 断点序号用 info breakpoints 获取.
- disable n -- n 为断点标识. 禁用第 n 个断点.
- enable n -- n 为断点标识. 启用第 n 个断点.
- run, r: 运行程序
- next, n: 单步调试. 不进人函数调用, 单步执行.
- step, s: 进人函数内部, 单步执行.
- continue, c [跳过的断点数量]: 连续执行. 可以指定跳过多少个断点.
- set variable \<var\>=\<value\>: 设置变量值.
- backtrace, bt [full]: 显示调用堆栈. full -- 查看详细调用栈信息.
- frame n -- 进人 backtrace 命令显示的 "#n" 的栈桢. 默认显示当前函数名, 函数入参, 当前运行处所在源文件的代码行位置，并显示当前行代码.
- info, i \<信息类型\>: 显示信息. help info -- 可以查看 info 的帮助. 常用信息类型:
  - args -- 参数信息
  - locals -- 当前堆栈的变量.
  - variables -- 全局变量, 静态变量信息.
  - breakpoints, b -- 断点信息. Enb 字段表明断点启用(y)还是禁用(n).
  - watch -- 列出监视点和断点.
  - goroutines -- goroutines 列表.
- print, p \<参数\>: 打印变量或者其他信息(指针或地址前需要加 '*'). 参数为变量名 -- 打印变量信息. 参数为 \$NUM -- 打印历史中的倒数第 NUM 个变量. $ 代表倒数第一个, \$\$ 代表倒数第二个. \$len(v1) -- 变量 v1 的长度. \$cap(v1) -- 变量 v1 的容量.
- whatis \<变量名\>: 显示变量类型
- watch 表达式 -- 监视表达式的变化, 表达式的值从 0 (假值) 变为 1 (真值)时中断. 如果表达式是变量, 则变量变化时中断;

## 更多功能

### TUI 模式

汇编 + 代码 窗口模式. 快捷键: Ctrl + x + a
(gdb) layout split: 分割窗口, 同时显示代码和汇编.
