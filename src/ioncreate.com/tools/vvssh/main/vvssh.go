package main

import (
	"os"
	"io/ioutil"
	"strings"
	"strconv"
	"fmt"
	"ioncreate.com/tools/vvssh/utils"
	"github.com/spf13/pflag"
	"errors"
	"ioncreate.com/tools/vvssh/plugin"
	"bufio"
	"regexp"
	"io"
)

/*
以下在我41岁第一天时思路变化，：
我要创造一个简单的语言v，用来控制SSH，：

语言特征为：每行一个语句. 命令或者变量特征为$$开头空格结尾


<< 表示向SSH 命令输出 空格 后面跟命令 是内部命令 可以有$$INPUT（表示命令行输入），$$PASSWORD（表示输入密码），$$STR (表示后面的全部输出字符串) 
<< cd ls
<< $STR faf afa af fa f af

>> 表示本地向SSH输入 若后面有字符 可以字符串直接传递是或命令发送，若后面没有则表示终端输入，表示进入本地输入状态 这时SSH终端将被本地输入接管，本地输入的所有直接发送到SSH上

内部命令表

$$STR=XXXX (表示后面的全部输出字符串 比如： $$STR=\001\002)
$$PASSWORD（表示输入密码）
$$USER (表示输入用户名)


以上不算数，需要重新设计


*/

func PrintTitle() {
	/*
	本工具仅仅,用于不用频繁敲打密码，在远程服务器执行维护安装命令使用,
	使用方法：
	No simple Command:
	vsshcmd [Remote Command File]

	vsshcmd [ConfigFile] [Remote Command File]

	vsshcmd [-u 用户名] [-p 密码] [-s 服务器] [-t 超时] [ConfigFile] [Remote Command File]


	has simple Command：

	vsshcmd [-u 用户名] [-p 密码] [-s 服务器] [-c 表示直接命令] [Simple Commands]...



	Remote Command File：
	同远程Shell文件一致，但是可以通过特殊指令执行本地命令
	若该行开始字符为#==之后的字符串将作为本地命令进行处理，以空格分割规则同SHELL脚本

	Plugins命令：
	scp 	[本地文件名/或目录名] [远程文件路径] 		执行SCP传送任务
	user		[用户名]									设置SSH用户名称
	server	[服务器地址] 								设置服务器地址 192.168.1.1:22
	pass 	[密码] 									设置SSH登陆密码

	autosudo										设置自动提供远程SUDO密码，同pass设置
	var_cmd	$[本地变量名称] [远端命令]					将远端命令执行后的结果存入本地变量
	var_env $[本地变量名称] [环境变量名称]              将环境变量存入本地变量，不会进行检查，运行时没有环境变量将报错
	*TODO var_chk $[本地变量名称]                            检查本地变量是否存在

	timeout	[时间]   								设置超时(秒)，默认30秒,超时从开始读取控制台输出计算，到读取到控制台输出行，规则是超时后首先发送\r\n 再次超时中断，执行后重新计算


	*TODO local 	[本地Shell文件名] 						执行本地shell文件 使用sh
	*TODO localrun	[本地执行文件]						运行本地文件 直接运行
	*TODO hostmode	[strict,ignore,fixed] [fixed key]	主机验证模式，严格模式，必须校验主机的安全信息
	*TODO? var_def	[本地变量名称] [变量字符串]					定义本地变量
	*??TODO reconnect									等待重新连接SSH，继续执行 ，其实也可以在脚本中来搞



	本地变量：
	本地变量使用：$[变量名称]
	内置变量：
	$[SYS_RUN_DIR]	当前vsshcmd运行目录


	配置文件变量：
	在常规配置文件项目后
	 [user] [password] [Server Address] [Time out]
	可以自行添加更多
	varName=var 的变量，空格区分


	*TODO 流程控制指令
	*TODO if_var [本地变量] ["字符串"]|[null|not_null|regex] [正则表达式]
	*TODO if_else
	*TODO if_end


	WORK:
	1. 重新梳理架构，弄一个Plugin的模型 DONE
	2. 提供整合SCP的功能 DONE


	Arch:

	main->init->Parse Params->CheckCMD->RUN


	参数测试优先级，0：命令行，1：Config File，2：Command File


	REF：
	https://github.com/mvdan/sh
	https://github.com/michaelmacinnis/oh

	*/
	fmt.Print(`
_____________________________________________
|                                           | 
|           <<VV's SSH Tools>>              |
|  It is a Good  SSH Automate Tools         |
|                         by Victor Ho      | 
|                         @My birthday,2018 |
|___________________________________________|
                             Version 0.7
`)

}

func PrintUsage() {

	PrintTitle()
	fmt.Print(`
Usage:
You can use with Command file:
vsshcmd [Remote Command File]
vsshcmd [Config File] [Remote Command File ]
vsshcmd [-u UserName] [-p Password] [-s Server] [-t TimeOut] [-a] [Config File] [Remote Command File]
Or just use simple Command：
vsshcmd [-u UserName] [-p Password] [-s Server] [-t TimeOut] [-a] [Simple Commands]...
Note: If you use flag as below, you setting option in [Config File] or [Remote Command File]  will be override.
`)
	pflag.CommandLine.SetOutput(os.Stdout)
	pflag.PrintDefaults()
}

func PrintFileUsage(){

	fmt.Print(`
=============================   File Usage  =======================================
1.Config File Format
Config File is simple UTF-8 text file.It is only one line:
UserName Password Server TimeOut
You can generate it just like this:
echo "root password 192.168.1.1:22 30" >Test.conf

2.Remote Command File Format

Remote Command File is just like shell script. We use "#==" as local command prefix.

====================================================================================
                         Happy Hack!
`)
}


type RemoteSSHCmd struct {
	CmdContent    []string
	RunCmdIdx     int
	CmdPlugins    map[string]plugin.RemoteSSHCmdPlugin
	SSH           *utils.SSHService
	GlobalCmdVars map[string]string
	globalReg *regexp.Regexp
}

func NewRemoteSSHCmd()*RemoteSSHCmd{
	SSHCmd:=RemoteSSHCmd{
		CmdContent:    []string{},
		RunCmdIdx:     0,
		CmdPlugins:    map[string]plugin.RemoteSSHCmdPlugin{},
		SSH:           utils.NewSSHService(),
		GlobalCmdVars: make(map[string]string),
		globalReg:regexp.MustCompile(`(\$\[)(\w+)(])`),
	}
	SSHCmd.installBuildInPlugins()
	return &SSHCmd
}

func CheckError(err error, msg string) {
	if err != nil {
		PrintTitle()
		fmt.Print(`
 ===============ERROR=======================
`)
		fmt.Printf("%s \nDetail:\n%v\n\n", msg, err)
		//PrintUsage()
		fmt.Print("Please use -h or --help flag to check usages!\n")
		os.Exit(1)
	}
}

func(rsc *RemoteSSHCmd)installBuildInPlugins() {
	/*
	scp 	[本地文件名/或目录名] [远程文件路径] 		执行SCP传送任务
	user		[用户名]									设置SSH用户名称
	server	[服务器地址] 								设置服务器地址 192.168.1.1:22
	pass 	[密码] 									设置SSH登陆密码
	timeout	[时间]   								设置超时(秒)，默认30秒
	autosudo										设置自动提供远程SUDO密码，同pass设置
	 */

	userPlugin:=plugin.RemoteSSHCmdPlugin{
		Name:"user",
		Check: func(context *plugin.SSHContext, commands []string) error {
			if len(commands)!=1 {
				return errors.New("user set error")
			}
			rsc.SSH.User=commands[0]
			return nil
		},
		Run:nil,
		Unload:nil,
		Context:nil,
	}
	rsc.CmdPlugins[userPlugin.Name]=userPlugin

	serverPlugin:=plugin.RemoteSSHCmdPlugin{
		Name:"server",
		Check: func(context *plugin.SSHContext, commands []string) error {
			if len(commands)!=1 {
				return errors.New("server set error")
			}
			serverParts:= strings.Split(commands[0],":")
			if len(serverParts)!=2{
				return errors.New("server set error! must like this:127.0.0.1:22 ")
			}
			if _,err:=strconv.Atoi(serverParts[1]);err!=nil{
				return errors.New("server set error! must like this:127.0.0.1:22")
			}
			rsc.SSH.Server=commands[0]
			return nil
		},
		Run:nil,
		Unload:nil,
		Context:nil,
	}
	rsc.CmdPlugins[serverPlugin.Name]=serverPlugin

	passPlugin:=plugin.RemoteSSHCmdPlugin{
		Name:"pass",
		Check: func(context *plugin.SSHContext, commands []string) error {

			rsc.SSH.Password=strings.Join(commands," ") //note: must same as splite
			return nil
		},
		Run:nil,
		Unload:nil,
		Context:nil,
	}
	rsc.CmdPlugins[passPlugin.Name]=passPlugin


	timeoutPlugin:=plugin.RemoteSSHCmdPlugin{
		Name:"timeout",
		Check: func(context *plugin.SSHContext, commands []string) error {

			if len(commands)!=1 {
				return errors.New("timeout set error")
			}

			if t,err:=strconv.Atoi(commands[0]);err!=nil{
				return errors.New("time set error! must be a number like: 100 ")
			}else{
				if rsc.SSH.TimeOut!=t{
					rsc.SSH.TimeOut=t
					rsc.SSH.StopWatchDog()
					rsc.SSH.EnableWatchDog()
				}

			}
			return nil
		},
		Run:nil,
		Unload:nil,
		Context:nil,
	}
	rsc.CmdPlugins[timeoutPlugin.Name]=timeoutPlugin

	autosudoPlugin:=plugin.RemoteSSHCmdPlugin{
		Name:"autosudo",
		Check: func(context *plugin.SSHContext, commands []string) error {
			rsc.SSH.IsAutoSudo=true
			return nil
		},
		Run:nil,
		Unload:nil,
		Context:nil,
	}
	rsc.CmdPlugins[autosudoPlugin.Name]=autosudoPlugin

	scpPlugin:=plugin.ScpCmdRegisterPlugin()
	rsc.CmdPlugins[scpPlugin.Name]=scpPlugin


	varcmdPlugin:=plugin.VarCmdRegisterPlugin()
	rsc.CmdPlugins[varcmdPlugin.Name]=varcmdPlugin


	varenvPlugin:=plugin.VarEnvRegisterPlugin()

	rsc.CmdPlugins[varenvPlugin.Name]=varenvPlugin

	varChkPlugin:=plugin.RemoteSSHCmdPlugin{
		Name:"var_chk",
		Check: func(context *plugin.SSHContext, commands []string) error {
			if !strings.HasPrefix(commands[0],"$[")||
				!strings.HasSuffix(commands[0],"]"){
				return errors.New("line var format error! must like $[var_name]")
			}
			varName:=strings.TrimLeft(commands[0],"$[")
			varName=strings.TrimRight(varName,"]")
			if _,find:=rsc.GlobalCmdVars[varName];!find{
				return errors.New("can't find var error! var name: "+varName)
			}
			return nil
		},
		Run:nil,
		Unload:nil,
		Context:nil,
	}

	rsc.CmdPlugins[varChkPlugin.Name]=varChkPlugin

	for _,v:=range rsc.CmdPlugins{
		if v.Context!=nil{
			v.Context.VarRegex=rsc.globalReg
			v.Context.Vars=&rsc.GlobalCmdVars
			v.Context.SSHConsoleReadToPromote=rsc.SSH.ReadStrToShellPromote
			v.Context.SSHConsoleWriteCommand=rsc.SSH.WriteCommandToSSHShell
			v.Context.SetLineVars=rsc.ProcessLineVars
		}
	}


}

func(rsc *RemoteSSHCmd) ParseSimpleCommand()error {
	return nil
}


func(rsc *RemoteSSHCmd) ParseConfigFile(fileName string)error {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return errors.New(fileName+" is not exist!")
	}

	ConfigContent,err:=ioutil.ReadFile(fileName)
	ConfigContents:=strings.Split(string(ConfigContent),"\n")

	CheckError(err,"Can't Read Config File! ")

	ConfigStr:=ConfigContents[0]

	ConfigArgStrs:=strings.Split(ConfigStr," ")
	if len(ConfigArgStrs)<4 {
		CheckError(errors.New("ConfigFile Format:\n [user] [password] [Server Address] [Time out] [VarName=Var]..."),"Config File Format Error!")
	}


	fmt.Printf("ALL:%v\n",ConfigArgStrs[2])
	rsc.SSH.User=ConfigArgStrs[0]
	rsc.SSH.Password=ConfigArgStrs[1]
	rsc.SSH.Server=ConfigArgStrs[2]
	TimeOut,err:=strconv.Atoi(ConfigArgStrs[3])
	if err==nil{
		rsc.SSH.TimeOut=TimeOut
	}else{
		CheckError(err,"Config File Error at TimeOut,\nConfigFile Format:\n [user] [password] [Server Address] [Time out] [VarName=Var]...")
	}
	if  len(ConfigArgStrs)>4{
		Vars:=ConfigArgStrs[4:]
		for idx:=range Vars{
			varAll:=Vars[idx]
			varArgs:=strings.Split(varAll,"=")
			if len(varArgs)!=2{
				CheckError(errors.New("Config Var must like this: varName=var \n ConfigFile Format:\n [user] [password] [Server Address] [Time out] [VarName=Var]..."),"Config File Var Error!")
			}else{
				rsc.GlobalCmdVars[varArgs[0]]=varArgs[1]
			}
		}
	}
	fmt.Print("Config File:"+fileName+" ok!")

	return nil
}

func(rsc *RemoteSSHCmd) ParseCommandFile(fileName string)error {


	CheckLine:=func(c string) (LocalCMD string, isAvailable bool,err error) {

		if strings.Index(c, "#") == 0 {
			if strings.Index(c, "#==") == 0 {
				AllCMDStr := strings.TrimLeft(c, "#==")
				AllCMDStrs := strings.Split(AllCMDStr, " ")
				findCMD := strings.ToLower(AllCMDStrs[0])
				if itPlugin, find := rsc.CmdPlugins[findCMD]; !find {
					return findCMD, false, errors.New("Can't find any plugin for Local Command:" + AllCMDStrs[0])
				} else {
					if err := itPlugin.Check(itPlugin.Context, AllCMDStrs[1:]); err != nil {
						return findCMD, false, fmt.Errorf("plugin for Local Command: %s Check fail!\n%s", AllCMDStrs[0], err.Error())
					}

					if itPlugin.Run!=nil{
						return findCMD, true, nil
					}
					return findCMD, false, nil
				}
			}
			return "", false, nil
		}
		return "", true, nil
	}



	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return errors.New(fileName+" is not exist!")
	}



	vCMDFileContent,err:=ioutil.ReadFile(fileName)
	CheckError(err,"Can't Remote Command File! ")

	vCMDFileCmds:=strings.Split(string(vCMDFileContent),"\n")
	for idx := range vCMDFileCmds {
		lineStr := strings.TrimSpace(vCMDFileCmds[idx])
		if lineStr ==""{
			// it empty line
			continue
		}
		if funcStr,isAvailable,err:=CheckLine(lineStr);err!=nil{
			return fmt.Errorf("proecess Remote Command File Error At Line:%d\t Unknow Function Name or val:%s\r\nError:%s", idx,funcStr,err.Error())
		}else if isAvailable{
			rsc.CmdContent=append(rsc.CmdContent, lineStr)
		}//else is a comment
	}

	return nil


}


func (rsc *RemoteSSHCmd) setLastOverrideArgs(user *string,pass *string,server *string,timeout *int,isAutoSudo *bool,isMustBeDone bool)error{
	if(*user)!=""{
		rsc.SSH.User=*user
	}else if isMustBeDone|| rsc.SSH.User==""{

		return errors.New("user have not be set")
	}

	if(*pass)!=""{
		rsc.SSH.Password=*pass
	}else if isMustBeDone|| rsc.SSH.Password==""{
		return errors.New("password have not be set")
	}

	if(*server)!=""{
		rsc.SSH.Server=*server
	}else if isMustBeDone|| rsc.SSH.Server==""{
		return errors.New("server have not be set")
	}


	if(*timeout)!=-1{
		rsc.SSH.TimeOut=*timeout
	}else {
		if rsc.SSH.TimeOut==0{
			rsc.SSH.TimeOut=30
		}
	}


	if rsc.SSH.User=="root"{
		rsc.SSH.ShellPromoteByte ='#'
	}else{
		rsc.SSH.ShellPromoteByte ='$'
	}

	if *isAutoSudo&&!rsc.SSH.IsAutoSudo{
		rsc.SSH.IsAutoSudo=true
	}

	return nil
}


func(rsc *RemoteSSHCmd) CheckArgs(user *string,pass *string,server *string,timeout *int, isSimpleCmd *bool,isAutoSudo *bool)error {
	NoFlagCmds:=pflag.Args()
	NoFlagCmdCount:=len(NoFlagCmds)
	if !*isSimpleCmd {
		switch NoFlagCmdCount {
		case 1:
			err:= rsc.ParseCommandFile(NoFlagCmds[0])
			CheckError(err,"Parse Command File")
		case 2:
			err:= rsc.ParseConfigFile(NoFlagCmds[0])
			CheckError(err,"Parse Config File")
			err= rsc.ParseCommandFile(NoFlagCmds[1])
			CheckError(err,"Parse Command File")
			err= rsc.setLastOverrideArgs(user,pass,server ,timeout,isAutoSudo,false)
			CheckError(err,"Parse Files: some args must be set!")
		default:
			return fmt.Errorf(" Check Args Error")
		}
	}else{
		println("Simple Command Mode!")
		err:= rsc.setLastOverrideArgs(user,pass,server ,timeout,isAutoSudo,true)
		CheckError(err,"Parse Simple Commands: The flag must be set!")
		rsc.CmdContent=NoFlagCmds
		err= rsc.ParseSimpleCommand()
		CheckError(err,"Parse Simple Commands")
	}

	return nil
}

func (rsc *RemoteSSHCmd) StartLocalInputService(){

}


func (rsc *RemoteSSHCmd) StopLocalInputService(){

}


func(rsc *RemoteSSHCmd) ProcessLineVars(line string)(string) {
	results := rsc.globalReg.FindAllStringSubmatch(line, -1)
	resultStr:=line
	for idx:=range results{
		if _,find:=rsc.GlobalCmdVars[results[idx][2]];find{
			resultStr=strings.Replace(resultStr,results[idx][0],rsc.GlobalCmdVars[results[idx][2]],-1)
		}
	}
	return resultStr
}

func(rsc *RemoteSSHCmd) RunCommands()error{

	RunLine:=func(c string) error {
		CMDStr := strings.TrimLeft(c, "#==")

		AllCMDStrs := strings.Split(CMDStr, " ")
		findCMD := strings.ToLower(AllCMDStrs[0])
		if itPlugin, find := rsc.CmdPlugins[findCMD]; !find {
			return errors.New("Runtime Can't find any plugin for Local Command:" + AllCMDStrs[0])
		} else if itPlugin.Run!=nil {
			if err := itPlugin.Run(itPlugin.Context, AllCMDStrs[1:]); err != nil {
				println("start plugin Err.."+err.Error())
				return fmt.Errorf("Runtime plugin run Local Command: %s fail! Error:\n%s", AllCMDStrs[0], err.Error())
			}else{
				return nil
			}
		}else {
			return errors.New("Runtime have not any run :" + AllCMDStrs[0])
		}
	}

	IOReader:=bufio.NewReader(rsc.SSH.RemoteMainIO.Out)
	rsc.updatePluginsContext(IOReader,rsc.SSH.RemoteMainIO.In)
	err:= rsc.SSH.ReadStrToShellPromote(IOReader,nil)
	if err!=nil{
		return err
	}
	//println(" Cmd Start!")
	for idx := range rsc.CmdContent {
		cmdStr := rsc.CmdContent[idx]
		rsc.RunCmdIdx=idx
		//println("current Cmd:",cmdStr)

		if strings.Index(cmdStr, "#==")==0 {
			err:=RunLine(cmdStr)
			if err!=nil{
				return err
			}
		} else {
			cmdStr=rsc.ProcessLineVars(cmdStr)
			err := rsc.SSH.WriteCommandToSSHShell(IOReader,cmdStr)
			//println(" Cmd Write:", cmdStr)
			if err != nil {
				return err
			}
			//println(" Cmd Start Wait Result:", cmdStr)
			err = rsc.SSH.ReadStrToShellPromote(IOReader,nil)
			if err != nil {
				return err
			}

		}
	}
	rsc.SSH.WriteCommandToSSHShell(IOReader,"exit")
	return nil
}



func(rsc *RemoteSSHCmd) WaitSSHEnd()error{
	err:= rsc.SSH.WaitMainSession()

	return err
}

func(rsc *RemoteSSHCmd) updatePluginsContext(reader *bufio.Reader,writer io.Writer){
	//DONE::UpdatePluginsContext
	for _,Item:=range rsc.CmdPlugins{
		if Item.Context!=nil{
			Item.Context.SSHClient= rsc.SSH.Client
			Item.Context.Vars=&rsc.GlobalCmdVars
			Item.Context.SSHConsoleReader=reader
			Item.Context.SSHConsoleWriter=writer
			//Item.Context.CloseChan=rsc.SSH.MainSessionCloseChan
		}
	}

}




func(rsc *RemoteSSHCmd) Run()error{
	rsc.SSH.Start()
	//err:=rsc.SSH.ReadStrToShellPromote()
	//if err!=nil{
	//	return err
	//}

	//println("SSH Start")

	err:= rsc.RunCommands()
	CheckError(err,"RunCommands")
	//rsc.WaitSSHEnd()
	return rsc.SSH.Stop()
}





func main(){

	//处理配置文件
	var User = pflag.StringP("user","u","","User Name of SSH")
	var Password =pflag.StringP("password","p","","Password of SSH")
	var Server=pflag.StringP("server","s","","Server Address like 127.0.0.1:22")
	var TimeOut=pflag.IntP("timeout","t",-1,"Set Timeout of Commands. Default:30 secs")
	var IsAutoSudo=pflag.BoolP("autosudo","a",false,"Set auto sudo for ssh session")
	var IsSimpleCmd =pflag.BoolP("command","c",false,"Set Command from CLI")
	var IsHelp =pflag.BoolP("help","h",false,"Help like this")
	pflag.Parse()
	if *IsHelp{
		PrintUsage()
		PrintFileUsage()
		return
	}

	SSHCmd:=NewRemoteSSHCmd()



	err:=SSHCmd.CheckArgs(User,Password,Server,TimeOut, IsSimpleCmd,IsAutoSudo)
	CheckError(err,"Check Command Args")

	err=SSHCmd.Run()
	CheckError(err,"Run")

}
