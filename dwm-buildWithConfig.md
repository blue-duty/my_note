## [dwm](https://dwm.suckless.org/)

> dynamic window manager是X上的一个动态窗口管理器

用让我认识dwm的博主的话来说，dwm是一个“简单的，快速的，可定制的窗口管理器”。dwm的设计目标是快速，轻量，可定制，而不是功能齐全。dwm的代码量只有1000行左右，而且是用C语言写的，所以它的运行速度非常快。dwm的配置文件也很简单，只有一个config.h文件，里面的宏定义就是dwm的配置。还可以通过自己编写的patch来定制dwm的功能。

### 为什么选择dwm

用我主观的想法来说，dwm让我的Linux变得更酷了。也让我的电脑更快了，说个题外的话，我前天晚上突发奇想打开了我电脑上的另一个系统（Windows11），开机到我能在桌面右键菜单，大概用了6-7分钟，但也可能由于是好久没有使用后的第一次打开的原因。而我使用Manjaro,基本在10秒内可以搞定，这也我一直想换电脑却一直没有换的原因之一。其次，纯c写的有什么好处呢，我能看得懂，我甚至在我需要什么功能的时候看一下源码，然后就可以自己去自定义，然后一行简单的命令`sudo make clean install`重启后就可以使用了。我之前使用了Manjaro xfce将近1年左右，使用体验还是很美好的，强大的yay造就了无限的可能，至于使用dwm纯粹是在b站上看到很炫酷产生了好奇，以及在xfce上始终没有找到类似于windows的窗口分割的功能。于是，开启了dwm之旅。


### 安装


1. 如果还没有安装xserver环境，先安装xserver环境

    ```bash
    sudo pacman -S xorg-xinit
    ```
2. clone dwm源码

    ```bash
    git clone https://git.suckless.org/dwm
    ```
    这一步可能会很慢，因为毕竟是国外的服务器，当然，我们可以选择github上个人定制话的源码，也可以选择拉取我借鉴的源码，这样就可以省去这一步了。

    ```bash
    git clone https://github.com/blue-duty/dwm.git
    ```
3. make

    ```bash
    cd dwm
    sudo make clean install
    ```
    这一步会将dwm安装到/usr/local/bin目录下，如果你想安装到其他目录，可以使用`sudo make clean install PREFIX=/usr`，这样就会安装到/usr/bin目录下。
4. 安装字体

    ```bash
    yay -S wqy-microhei
    yay -S wps-office-mui-zh-cn
    yay -S ttf-wps-fonts
    yay -S nerd-fonts-jetbrains-mono
    yay -S ttf-material-design-icons
    yay -S ttf-joypixels
    yay -S ttf-dejavu
    ```
5. 快速体验

    ```bash
    echo "exec dwm" > ~/.xinitrc
    startx
    ```

    如果你想使用dwm，那么你就需要在你的~/.xinitrc文件中添加`exec dwm`，然后在tty（Ctrl+Alt+F2，因为你当前正在tty1，所以可以选择tty2之后的tty）中使用`startx`命令启动dwm。

6. 将dwm添加到窗口显示管理器中

    Manjaro默认使用的是lightdm，所以我们需要将dwm添加到lightdm中，这样才能在登录界面选择dwm。

    在 /usr/share/xsessions/ 目录下创建一个dwm.desktop文件，内容如下：

    ```bash
    [Desktop Entry]
    Encoding=UTF-8
    Name=dwm
    Comment=dynamic window manager
    Exec=/usr/local/bin/dwm
    Icon=dwm
    Type=XSession
    ```

    然后重启就可以在登录界面选择dwm了。



