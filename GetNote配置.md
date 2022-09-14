# 从头配置一个Getnote环境

## 一、安装gitnote
不过多赘述，自己在官网下载即可，不过提一点，我用的github，仓库无法拉取下来，只能自己下载下来后绑定仓库。

> 后续：由于GitHub新建仓库默认只有main分支，而gitnote默认拉取master分支，所以无法拉取成功，解决方法：新建一个master分支，或者直接将默认的main分支改为master。

## 二、配置插件
由于未知因素可能导致插件无法下载成功，我们可以选择通过作者的[插件仓库]()将插件下载到本地，然后解压`{user}/.gitnote/plugins`中。

### 配置图床
由于自带的github图床永远无法成功，可以使PicGo+GitHub+jsdelivr配置图床，具体操作如下：
1. 在Github创建一个用于存放图片的仓库。
2. 创建一个访问令牌，具体权限如下：![](https://gcore.jsdelivr.net/gh/blue-duty/gitnote-images/img/20220915005550.png)
3. 配置PicGo，如图，仓库名和分支名按实际情况填写，Token即为上步设置的访问令牌，存储路径按自己想法填写，**自定义域名为`https://gcore.jsdelivr.net/gh/{git用户名}/{仓库名}`**![](https://gcore.jsdelivr.net/gh/blue-duty/gitnote-images/img/20220915005700.png)
4. 设置完成后，就通过PicGo上传图片。

## 三、Git更新仓库
目前来看，GitNote

