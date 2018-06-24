package plugin

import (
	"errors"
	"strings"
)

/*

Plugins命令：
var_cmd	[本地变量名称] [远端命令]					将远端命令执行后的结果存入本地变量
 */
func VarCmdRegisterPlugin()RemoteSSHCmdPlugin{
	res:=RemoteSSHCmdPlugin{
		Name:   "var_cmd",
		Check:  varCmdCheck,
		Run:    varCmdRun,
		Unload: nil,
		Context:&SSHContext{
			Vars:nil,
			SSHClient:nil,
		},
	}
	return res
}

func varCmdCheck(context *SSHContext,commands []string)error{
	if len(commands)<2 {
		return errors.New("too few args")
	}

	if !strings.HasPrefix(commands[0],"$[")||
		!strings.HasSuffix(commands[0],"]"){
			return errors.New("line var format error! must like $[var_name]")
	}

	cmdStr:=strings.Join(commands[1:]," ")

	if HasVars(context,cmdStr){
		return errors.New("var_cmd can't use Vars In Remote Command")
	}

	return nil
}

func varCmdRun(context *SSHContext,commands []string)error{

	println("start var_cmd..")
	varName:=strings.TrimLeft(commands[0],"$[")
	varName=strings.TrimRight(varName,"]")
	cmdStr:=strings.Join(commands[1:]," ")
	ResultStr:=""
	context.SSHConsoleWriteCommand(context.SSHConsoleReader,cmdStr)
	context.SSHConsoleReadToPromote(context.SSHConsoleReader, func(line string) {
		if ResultStr==""{
			ResultStr+=strings.TrimRight(line,"\r\n")
		}else{
			ResultStr+="\n"+line
		}
	})
	println("process var_cmd.. Result=",ResultStr)
	(*context.Vars)[varName]  =ResultStr
	
	println("end var_cmd..")
	return nil
}
