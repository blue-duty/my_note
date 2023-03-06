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
NF 代表字段的个数，NR 代表行号。
printf() 代表格式化输出，printf("%s", $1) 代表输出第一个字段，printf("%s", $2) 代表输出第二个字段，以此类推。
输出排序：sort -n -k 1 -t "," 代表按照第一个字段进行排序，-n 代表按照数字进行排序，-k 1 代表按照第一个字段进行排序，-t "," 代表按照逗号进行分隔。
运算符：+ - * / % ++ -- += -= *= /= %= == != > < >= <= && || ! ~ ~=
逻辑运算符：&& || !

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




