# VV's SSH Tools



[TOC]

## English

### 1. What is it
Each time when I faced a release deployment, I faced countless numbers of physical or virtual PC devices and repeatedly tapped the keyboard and hit the "SSH" command. I am tired, I want to change. For this I learned [Expect](https://en.wikipedia.org/wiki/Expect) , which is too complicated. It also requires a lot of TK/TCL scripts to be installed on the system, which is too cumbersome. So I gave myself a birthday present and gave it to all of you.

**This is an automated tool for SSH remotely running scripts!** it can:
- Automatic  login SSH server by just wrtie  a simple configuration file
- Automatically run remote shell commands just like you type commands manually
- Automatically populated sudo password just like you entered it manually
- Automatically read remote command output just like you do it manually copying/pasting
- Built-in SCP functionality eliminates the need to retype command lines and passwords
- Other features, waiting for your contribution to join :-)
### 2. Usage
#### 2.1. Install

Download Release Here,And put it into you script folder. That's it!

|   OS    |  Arch  |
| :-----: | :----: |
| Windows |  X86   |
| Windows | X86-64 |
|  Linux  |  X86   |
|  Linux  | X86-64 |
|  Linux  | ARMv7  |
|  Linux  | ARMv8  |
|  MacOS  | X86-64 |
|  MacOS  |  X86   |



#### 2.2. Quick Start

#### 2.3. Commond Line Usage

You can run from command line interface like this:

- `vvssh [config file]` 
- `vvssh [chongfig file] [remote shell script file]`
- `vvssh [-u user] [-p pass] [-s ServerAddressAndPort] [-t timeout] [config file] [remote shell script file]`
- TODO：`vvssh [-u user] [-p pass] [-s ServerAddressAndPort] [-t timeout] [-c] [simple command]`



#### 2.4. Write a Config File

The configuration file is mainly used to store some remote host information, so as not to get you all day long, enter the password.
There is only one line in the configuration file. There are four fields, separated by spaces, which are the following:



```
[user] [password] [Server Address] [Time out]
```

| field          | Instructions             | example        |
| -------------- | ------------------------ | -------------- |
| user           | login name                   | admin          |
| password       | login password                  | thisispass     |
| Server Address | ssh server address and**port** | 192.168.0.1:22 |
| Time out       | timeout(sec) | 30             |

Here is an example🌰：

```
admin thisisPass 192.168.0.1:22 20
```

> PS:I usually use a shell file to generate a configuration file directly with `echo "root test1 192.168.0.1 30" >Server.conf`

#### 2.5. Write a Remote Shell Script

##### 2.5.1 Brief introduction
The remote script file is executed on the remote host just as if you were hitting the command directly in the terminal. But there are some very simple special rules you need to understand.

> PS:You can easily edit the remote script file using a syntax highlighting editor that supports shell scripts.

- Use `#==` as a marker for single-line commands

   > The `#` symbol is a comment symbol in a shell script,

- Tags using `$[]` as an internal variable

   > The reason I chose this tag is that it is rarely used in terminal shell commands at SSH session. However, this flag is ambiguous in many shell scripting environments and may be adjusted later. I wait your suggestions.


##### 2.5.2 Internal variables
Internal variables are used to store some temporary variable strings. Legal variables in the form `$[variable name]` will be replaced with the string corresponding to the variable. For example, the following command will obtain the current directory Path from the remote host,, then create a directory `ttt`, and then use `scp` copy the local file to the remote host:

```bash
#==var_cmd $[HOME_DIR] pwd
mkdir -p $[HOME_DIR]/ttt
#==scp /test1.file $[HOME_DIR]/ttt
```
Internal variables can also use environment variables to communicate between two different native shell scripts.

For example: There are two scripts are called `install.sh` and `remote-cmd.sh`

~~~bash
#!/usr/bin/env bash
#this is install.sh
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
RUNDIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
echo "Runing From $RUNDIR"
export PUBLISH_DIR=${RUNDIR}

echo "test pass 192.168.0.1 30" >${RUNDIR}/Server.conf

vvssh ${RUNDIR}/Server.conf remote-cmd.sh
~~~



~~~bash
#!vvssh
#this is remote-cmd.sh
#==autosudo

#Get the environment variable named “PUBLISH_DIR" from “install.sh"
#==var_env $[PUBLISH_DIR] PUBLISH_DIR

# Execute "pwd" command on remote host and store return string into "SSH_HOME_DIR" internal variable
#==var_cmd $[SSH_HOME_DIR] pwd

sudo apt-get -y -q install build-essential

mkdir $[SSH_HOME_DIR]/test

#==scp $[PUBLISH_DIR]/test.file $[SSH_HOME_DIR]/test

~~~


##### 2.5.3 Single-line command
The following commands are currently supported:
1. user 

   - **Description: **Set the SSH user name (yes! You can set it in the script file), this setting will override the previous command parameters and the settings of the configuration file

   - **Parameters: **`[user name]`

   - **Example:**

     ````bash
     #==user admin
     ````

     

2. server

   - **Description:** Set the ssh server address and port (yes! You can set it in the script file), this setting will override the previous command parameters and the settings of the configuration file

   - **Parameters:** `[Server Address And Port]`

   - **Example:** 

     ```bash
     #==server 192.168.1.1:22
     ```

     

3. pass

   - **Description:** Set SSH login password, (yes! You can set it in the script file), this setting will override the previous command parameters and configuration file settings

   - **Parameters:** `[Password]`

   - **Example:** 

     ```bash
     #==pass test123
     ```

     

4. timeout

   - ** Description: ** Set timeout (seconds). The default is 30 seconds. The timeout will be calculated from the start of reading the console output character line, to read the next output character line to the console, the rule is to first send `\r\n` after the timeout timeout interrupt again, recalculate after execution

   - **Parameters:**`[Time]`

   - **Example:** 

     ```bash
     #==timeout 30
     ```

     

5. autosudo

   - ** Description: ** Set the remote SUDO password automatically, using the password specified by the “pass” command parameter. After using this command, the password will be input automatically when the “sudo” command line password input prompt is encountered. Just as you type it on the keyboard.

   - **Parameters:** none

   - **Example:** 

     ```bash
     #==autosudo
     ```

     

6. scp

   - **Description:** Perform SCP file transfer task, can transfer directory and single file to remote target host

   - **Parameters: **`[local file name/or directory name] [remote file path]`

   - **Note:**

   - The scp command has not provided the ability to change the destination file (directory) name (save as)
   - The scp command must ensure that the path exists on the remote host

   - **Example:** 

     ```bash
     # Transfer a file to the /home/test directory on the remote host
     #==scp /test.file /home/test
     
     #Transfer the directory to the /home/test directory on the remote host
     #==scp /test1 /home/test
     ```

     

7. var_cmd

   - **Description: ** After the command is executed on the target host, the executed string result is stored in the local variable

   - **Parameters: **`$[local variable name] [remote command]`

   - **Example:**

     ```bash
     # Saves the string returned by the "pwd" command on the target host into the "HOME" variable
     #==var_cmd $[HOME] pwd
     ```

    ```bash
     #==var_cmd $[DIRS] ls -l
    ```

8. var_env

   - ** Description: ** Save local context variables into local variables

   - **Parameters:** `$[local variable name] [environment variable name]`

   - **Note: ** This command will not perform a pre-check of the existence of an environment variable. If there is no environment variable at runtime, an error will be reported.

   - **Example:**

     ```bash
     #==var_env $[LOCAL_RUN_DIR] LOCAL_DIR
     ```


### 3. How Contribution
TODO 

   


## 中文版

### 1. 这是个啥货？

每次当我面临发布部署时，面对无数的无论是实体的还是虚拟的PC设备，无数次反复敲打键盘，敲打着“SSH”命令。我累了，我要改变。为此我学习了[Expect ](https://en.wikipedia.org/wiki/Expect) ，这货太复杂。而且需要在系统上装一大堆TK/TCL脚本，太麻烦。所以我给我自己一个生日礼物，也给你们大家。



**这是一个SSH远程运行脚本的自动化工具！** 它可以：

- 自动ssh登录服务器，一个配置文件就搞定
- 自动运行远程shell命令，就像你在手工输入
- 自动填充sudo密码，就像你在手工输入
- 自动读取远端的命令输出，就像你在手工复制/粘贴
- 整合SCP功能，不用再反复输入密码和命令行
- 其他功能，等待你的贡献加入 :-)





从此不用再：

- `scp`,`ssh`一个个命令反复敲打
- 反复密码输入
- 远程多机部署配置，一个脚本轻松搞定



> 感谢以下项目给予的启示与帮助
>
> 

### 2. 这货的使用方法

#### 2.1. 安装

在[这里](https://github.com/iONCreate/vvssh/release/)下载对应的版本即可，放置在对应目录

| 操作系统 |  架构  |
| :------: | :----: |
| Windows  |  X86   |
| Windows  | X86-64 |
|  Linux   |  X86   |
|  Linux   | X86-64 |
|  Linux   | ARMv7  |
|  Linux   | ARMv8  |
|  MacOS   | X86-64 |
|  MacOS   |  X86   |

#### 2.2. 快速上手





#### 2.3. 命令行使用

按照一下方式运行

- `vvssh [远程脚本文件名]` 
- `vvssh [配置文件名] [远程脚本文件名]`
- `vvssh [-u 用户名] [-p 密码] [-s 服务器] [-t 超时] [配置文件名] [远程脚本文件名]`
- *待实现：`vvssh [-u 用户名] [-p 密码] [-s 服务器] [-t 超时] [-c] [简单脚本]`* 



#### 2.4. 配置文件编写说明

配置文件主要用于存储一些主机的信息，免得你一天到晚，输入密码。

配置文件只有一行，共四个字段，以空格间隔，为以下内容：

```
[user] [password] [Server Address] [Time out]
```

| 字段           | 说明                     | 例子           |
| -------------- | ------------------------ | -------------- |
| user           | 用户名                   | admin          |
| password       | 密码                     | thisispass     |
| Server Address | ssh 服务器地址与**端口** | 192.168.0.1:22 |
| Time out       | 超时时间设置（单位：秒） | 30             |

以下是一个例子🌰：

```
admin thisisPass 192.168.0.1:22 20
```

> PS: 我一般都是用一个Shell 文件套用，用`echo "root test1 192.168.0.1 30" >Server.conf` 这样的形式直接生成一个我配置文件



- 未来可以支持多行不同的配置，看看大家的要求投票吧



#### 2.5. 远程脚本编写说明

##### 2.5.1. 简介

远程脚本文件在远程主机上执行，就像你在终端中直接敲打命令一样。但是有一些非常简单特殊规则你需要了解。

PS:你可以使用一个支持shell脚本的语法高亮编辑器轻松编辑该文件。

- 使用`#==`作为单行命令的标记

  > `#` 符号是shell 脚本中的注释符号

- 使用`$[]`作为内部变量的标记

  > 我选这个标记的原因是，现在终端命令中很少用到这个，但这个标示在很多shell脚本环境中具有歧义，也许以后会调整，大家可以多提建议

##### 2.5.2. 内部变量

内部变量使用来存储一些临时的可变字符串，凡事使用`$[变量名]`形式的合法变量，都将被替换为变量对应的字符串，例如下列命令将从目标主机上取得当前目录绝对路径，然后建立目录，再将本地文件复制到目标主机上：

```bash
#==var_cmd $[HOME_DIR] pwd
mkdir -p $[HOME_DIR]/ttt
#==scp /test1.file $[HOME_DIR]/ttt
```

内部变量还可以使用环境变量用来在两个不同的本机Shell 脚本之间沟通。

比如：现有两个脚本分别叫做install.sh和remote-cmd.sh

~~~bash
#!/usr/bin/env bash
#this is install.sh
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
RUNDIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
echo "Runing From $RUNDIR"
export PUBLISH_DIR=${RUNDIR}

echo "test pass 192.168.0.1 30" >${RUNDIR}/Server.conf

vvssh ${RUNDIR}/Server.conf remote-cmd.sh
~~~



~~~bash
#!vvssh
#this is remote-cmd.sh
#==autosudo

#从"install.sh" 上下文中获取环境变量 "PUBLISH_DIR"
#==var_env $[PUBLISH_DIR] PUBLISH_DIR

#在远程主机上执行"pwd"命令 并存储返回字符串到"SSH_HOME_DIR" 内部变量中
#==var_cmd $[SSH_HOME_DIR] pwd

sudo apt-get -y -q install build-essential

mkdir $[SSH_HOME_DIR]/test

#==scp $[PUBLISH_DIR]/test.file $[SSH_HOME_DIR]/test

~~~




##### 2.5.3. 单行命令

目前支持以下命令：

1. user 

   * **说明：**设置SSH用户名称，（没错！你可以在脚本文件中设置），这个设置将覆盖之前命令参数以及配置文件的的设置

   * **参数：**`[用户名]`

   * **例子：**

     ````bash
     #==user admin
     ````

     

2. server

   - **说明：** 设置服务器地址，（没错！你可以在脚本文件中设置），这个设置将覆盖之前命令参数以及配置文件的的设置

   - **参数：** `[服务器地址端口]`

   - **例子：** 

     ```bash
     #==server 192.168.1.1:22
     ```

     

3. pass

   - **说明：**设置SSH登陆密码，（没错！你可以在脚本文件中设置），这个设置将覆盖之前命令参数以及配置文件的的设置

   - **参数：**`[密码]`

   - **例子：**

     ```bash
     #==pass test123
     ```

     

4. timeout

   - **说明：**设置超时(秒)。默认30秒。超时将从开始读取控制台输出字符行计算，到读取到控制台下一个输出字符行，规则是超时后首先发送`\r\n` 再次超时中断，执行后重新计算

   - **参数：**`[时间]` 

   - **例子：**

     ```bash
     #==timeout 30
     ```

     

5. autosudo

   - **说明：**设置自动提供远程SUDO密码，使用“pass”命令参数设置的密码。使用该命令后，遇到“sudo” 的命令行密码输入提示时，将自动输入密码。就如同你在键盘上输入的一样。

   - **参数：** 无

   - **例子：**

     ```bash
     #==autosudo
     ```

     

6. scp

   - **说明：**执行SCP 文件传输任务，可以传送目录和单个文件到远程目标主机

   - **参数：**`[本地文件名/或目录名] [远程文件路径]`

   - **注意：** 

    - scp 命令尚未提供更改目标文件（目录）名的功能（另存为）
    - scp 命令必须确保远程目标主机上路径存在

   - **例子：**

     ```bash
     # 传送单个文件到目标主机上的/home/test目录中
     #==scp /test.file /home/test
     
     #传送目录到目标主机上的/home/test目录中
     #==scp /test1 /home/test
     ```

     

7. var_cmd

   - **说明：**在目标主机上执行命令后，将执行后的结果存入本地变量

   - **参数：**`$[本地变量名称] [远端命令]`

   - **例子：**

     ```bash
     # 将目标主机上 "pwd" 命令返回的字符串 存入 "HOME" 变量中
     #==var_cmd $[HOME] pwd
     ```

     

8. var_env

   - **说明：**将环境变量存入本地变量，

   - **参数：** `$[本地变量名称] [环境变量名称]`

   - **注意：** 该命令不会进行预先检查环境变量是否存在，在运行时若没有环境变量将报错

   - **例子：**

     ```bash
     #==var_env $[LOCAL_RUN_DIR] LOCAL_DIR
     ```

     





## 参考

- [Shell Prompt](https://en.wikibooks.org/wiki/Guide_to_Unix/Explanations/Shell_Prompt)
- 

