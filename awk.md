# AWK

## 基本概念

### 1. 什么是 AWK

AWK 是一种编程语言，它是一种数据驱动的解释型语言，主要用于文本处理，特别是处理表格型的文本文件（如 `CSV` 格式）。

> pattern { action } 

pattern 和 action 之间用空格分隔，action 可以是一个或多个语句，多个语句之间用分号分隔。
pattern叫模式，action叫动作，一般情况下，模式在前面，动作用花括号括起来，多个动作用分号分隔。

### 2. 基本语法

$num 代表第几个字段，$0 代表整行，$1 代表第一个字段，$2 代表第二个字段，以此类推。在END中，$0 会打印文件的最后一行，$1 会打印最后一行的第一个字段，$2 会打印最后一行的第二个字段，以此类推。
print 代表输出，print $0 代表输出整行，print $1 代表输出第一个字段，print $2 代表输出第二个字段，以此类推。print "" 代表输出空行。
NF 代表字段的个数，NR 代表行号，FNR 代表文件的行号, FS 代表字段分隔符，RS 代表行分隔符。
printf() 代表格式化输出，printf("%s", $1) 代表输出第一个字段，printf("%s", $2) 代表输出第二个字段，以此类推。
输出排序：sort -n -k 1 -t "," 代表按照第一个字段进行排序，-n 代表按照数字进行排序，-k 1 代表按照第一个字段进行排序，-t "," 代表按照逗号进行分隔。
运算符：+ - * / % ++ -- += -= *= /= %= == != > < >= <= && || ! ~ ~=
逻辑运算符：&& || !

#### 内建变量

| 变量 | 说明 |
| --- | --- |
| $0 | 当前行 |
| ARGV | 命令行参数 |
| ARGC | 命令行参数个数 |
| FILENAME | 当前文件名 |
| FNR | 当前文件的行号 |
| FS | 字段分隔符, 默认是空格 |
| NF | 当前行的字段个数 |
| NR | 当前行号 |
| OFS | 输出字段分隔符, 默认是空格 |
| ORS | 输出行分隔符, 默认是换行符 |
| RS | 行分隔符, 默认是换行符 |
| OFMT | 数字格式, 默认是%.6g |
| RLENGTH | 匹配的字符串长度 |
| RSTART | 匹配的字符串开始位置 |
| SUBSEP | 数组下标分隔符, 默认是34 |

#### 内建函数

| 函数 | 说明 |
| --- | --- |
| atan2(y, x) | 反正切函数 |
| cos(x) | 余弦函数 |
| exp(x) | 指数函数 |
| int(x) | 取整函数 |
| log(x) | 对数函数 |
| rand() | 产生随机数 |
| sin(x) | 正弦函数 |
| sqrt(x) | 平方根函数 |
| srand() | 产生随机数种子 |
| index(s, t) | 返回字符串 t 在字符串 s 中的位置 |
| length(s) | 返回字符串 s 的长度 |
| match(s, r) | 返回字符串 r 在字符串 s 中的位置 |
| split(s, a, fs) | 将字符串 s 按照分隔符 fs 分割成数组 a |
| sprintf(fmt, ...) | 格式化输出 |
| sub(r, s, t) | 将字符串 t 中的第一个匹配字符串 r 替换为字符串 s |
| gsub(r, s, t) | 将字符串 t 中的所有匹配字符串 r 替换为字符串 s |
| substr(s, p, n) | 返回字符串 s 中从位置 p 开始的 n 个字符 |
| tolower(s) | 将字符串 s 转换为小写 |
| toupper(s) | 将字符串 s 转换为大写 |




BEGIN 代表在开始之前执行，END 代表在结束之后执行。

```bash
BEGIN { action }
pattern { action }
END { action }
```

变量：变量名=变量值 `$3 > 15 { tmp = tmp + 1}` 代表将第三个字段大于 15 的行数赋值给 tmp 变量。既可以存储数字，也可以存储字符串。
字符串拼接：`str1 = "hello" str2 = "world" str3 = str1 str4 = str1 str2` 代表将 str1 和 str2 拼接起来赋值给 str3，将 str1 和 str2 拼接起来赋值给 str4。
数组：`arr[1] = "hello" arr[2] = "world"` 代表将 hello 和 world 存储到数组中，数组的下标从 1 开始。

#### 流程控制

if 语句：if (condition) { action } else { action }
for 循环：for (i = 1; i <= 10; i++) { action }
while 循环：while (condition) { action }


#### 常用一行命令

```bash
# 打印第一列
awk '{print $1}' file
# 输出总行数
awk 'END {print NR}' file
# 打印第一列和第二列
awk '{print $1,$2}' file
# 打印第一列和第二列，用逗号分隔
awk -F ',' '{print $1,$2}' file
# 打印第10行
awk 'NR==10' file
# 打印每一行的最后一个字段
awk '{print $NF}' file
# 打印每一行的倒数第二个字段
awk '{print $(NF-1)}' file
# 打印最后一行的最后一个字段
awk '{ field = $NF} END {print field}' file
# 打印字段数多于3的行
awk 'NF>3' file
# 打印最后一个字段值为 1 的行
awk '$NF==1' file
# 打印字段数的总和
awk '{sum += NF} END {print sum}' file
# 打印包含Beth的行
awk '/Beth/' file
# 打印不包含Beth的行
awk '!/Beth/' file
# 打印具有最大值的第一个字段以及整行
awk 'BEGIN {max = 0} {if ($1 > max) {max = $1; line = $0}} END {print max, line}' file
# 打印行长度多于10的行
awk 'length($0) > 10' file
# 每一行加上行数
awk '{print NR, $0}' file
# 交换第一个字段和第二个字段并打印整行
awk '{tmp = $1; $1 = $2; $2 = tmp; print $0}' file
# 打印删除了第二个字段的整行
‘awk '{$2 = ""; print $0}' file
# 将每一行的字段逆序打印
awk '{for (i = NF; i > 0; i--) printf("%s ", $i); print ""}' file
# 打印每一行的所有字段值之和
awk '{sum = 0; for (i = 1; i <= NF; i++) sum += $i; print sum}' file
# 将所有行的字段值累加起来并输出最后的结果
awk '{for (i = 1; i <= NF; i++) sum += $i} END {print sum}' file
# 将每一个的每一个字段用它的绝对值替换
awk '{for (i = 1; i <= NF; i++) $i = ($i < 0) ? -$i : $i; print $0}' file


#### 常见模式

1. `BEGIN { action }` 代表在开始之前执行，`END { action }` 代表在结束之后执行。
2. `pattern { action }` 代表匹配模式，`action` 代表动作, `pattern` 为真时，执行 `action`。
3. `/pattern/ { action }` 代表匹配模式，`action` 代表动作, 字符串中包含 `pattern` 时，执行 `action`。
4. `pattern1, pattern2 { action }` 代表匹配模式，`action` 代表动作, `pattern1` 和 `pattern2` 之间的行(包括这两行)执行 `action`。

```bash
BEGIN { FS = "\t"# make tab the field separator
printf("%10s %6s %5s
%s\n\n",
"COUNTRY", "AREA", "POP", "CONTINENT")
}
{ printf("%10s %6d %5d
%s\n", $1, $2, $3, $4)
area = area + $2
pop = pop + $3
}
END
{ printf("\n%10s %6d %5d\n", "TOTAL", area, pop) }
```

#### 字符串匹配

1. `/pattern/` 代表匹配模式，`action` 代表动作, 字符串中包含 `pattern` 时，执行 `action`。
2. `!/pattern/` 代表匹配模式，`action` 代表动作, 字符串中不包含 `pattern` 时，执行 `action`。
3. `~ /pattern/` 代表匹配模式，`action` 代表动作, 字符串中包含 `pattern` 时，执行 `action`。
4. `!~ /pattern/` 代表匹配模式，`action` 代表动作, 字符串中不包含 `pattern` 时，执行 `action`。


<!-- 正则表达式 TODO -->


# SED

sed 是一种流编辑器，它是一个非交互式的文本编辑器，它一次处理一行内容，处理完成后，将编辑后的内容打印到标准输出。sed 主要用来自动编辑一个或多个文件；简化对文件的反复操作；编写转换程序等。

#### 常用选项
    
1. `-n`：使用安静(silent)模式。在一般 sed 的用法中，所有来自 STDIN 的资料一般都会被列出到萤幕上。但如果加上 -n 参数后，则只有经过sed 特殊处理的那一行(或者动作)才会被列出来。
2. `-e`：直接在指令列模式上进行 sed 的动作编辑；
3. `-f`：直接将 sed 的动作写在一个文件内， -f filename 则可以执行 filename 内的 sed 动作；
4. `-r`：正则表达式使用扩展语法；
5. `-i`：直接修改读取的文件内容，而不是由萤幕输出。
6. `-c`：使用指令计数器，例如：sed -n '3,5p' file，这个命令会打印第三行到第五行，如果加上 -c 参数，则会显示出来的行数前面会标上行号。
7. `-i.bak`：直接修改读取的文件内容，而不是由萤幕输出。并且在修改文件之前，先备份文件，备份文件的扩展名为 .bak。

#### 位置参数

1. `n`：代表第 n 行。
2. `n, m`：代表第 n 行到第 m 行。
3. `n, $`：代表第 n 行到最后一行。
4. `n, +m`：代表第 n 行到第 n+m 行。
5. `n, -m`：代表第 n 行到第 n-m 行。
6. `/pattern/`：代表匹配模式，`action` 代表动作, 字符串中包含 `pattern` 时，执行 `action`。
7. `! /pattern/`：代表匹配模式，`action` 代表动作, 字符串中不包含 `pattern` 时，执行 `action`。
8. `n, /pattern/`：代表匹配模式，`action` 代表动作, 第 n 行到匹配 `pattern` 的行(包括这两行)执行 `action`。
9. `n, ! /pattern/`：代表匹配模式，`action` 代表动作, 第 n 行到不匹配 `pattern` 的行(包括这两行)执行 `action`。
10. `n, +m, /pattern/`：代表匹配模式，`action` 代表动作, 第 n 行到第 n+m 行中匹配 `pattern` 的行(包括这两行)执行 `action`。
11. `/pattern/, /pattern/`：代表匹配模式，`action` 代表动作, 匹配 `pattern` 的行到匹配 `pattern` 的行(包括这两行)执行 `action`。
12. `n~m`：代表每隔 m 行执行一次 `action`。

#### 编辑命令

1. `a`：在下一行添加文本。
2. `i`：在上一行添加文本。
3. `c`：替换 n 行的文本。
4. `d`：删除 n 行的文本。
5. `p`：打印 n 行的文本。
6. `w`：将 n 行的文本写入文件。
7. `r`：读取文件内容，并将内容插入到 n 行之后。
8. `=`：显示行号。
9. `!`：取反，与其他命令组合使用。
10. `s`：替换命令，格式为：`s/old/new/`，`s/old/new/g`，`s/old/new/2`，`s/old/new/g2`。
    - `s/old/new/`：将第一次出现的 `old` 替换为 `new`。
    - `s/old/new/g`：将所有出现的 `old` 替换为 `new`。
    - `s/old/new/2`：将第二次出现的 `old` 替换为 `new`。
    - `s/old/new/g2`：将第二次出现的 `old` 替换为 `new`。
    - `s/old/L&/g`：将 `old` 替换为 `Lold`。
    - `s/old/L\&/g`：将 `old` 替换为 `L&`。
    - `\l`：将下一个字符转换为小写。
    - `\u`：将下一个字符转换为大写。
    - `\L`：将剩余的字符转换为小写。
    - `\U`：将剩余的字符转换为大写。
    - `\E`：结束 `\L` 和 `\U` 的作用。
    - `\t`：制表符。
    - `\n`：换行符。
    - `\r`：回车符。
    - `\f`：换页符。
    - `\v`：垂直制表符。
    - `\a`：警报符。
    - `\e`：转义符。
    - `\c`：取消命令的执行。
    - `\d`：匹配一个数字。
    - `\D`：匹配一个非数字。
    - `\s`：匹配一个空白字符。
    - `\S`：匹配一个非空白字符。
    - `\w`：匹配一个单词字符。
    - `\W`：匹配一个非单词字符。
    - `\x`：匹配一个十六进制数。
    - `\0`：匹配一个八进制数。
11. `y`：转换命令，格式为：`y/old/new/`，`y/old/new/g`，`y/old/new/2`，`y/old/new/g2`。

#### 常用命令

```bash
# 打印第一行
sed -n '1p' file
# 打印第一行和第二行
sed -n '1,2p' file
# 打印第一行到第三行
sed -n '1,3p' file
# 打印第一行到最后一行
sed -n '1,$p' file
# 打印第一行到倒数第二行
sed -n '1,-2p' file
# 打印第一行和倒数第二行
sed -n '1p; -2p' file
# 打印包含Beth的行
sed -n '/Beth/p' file



