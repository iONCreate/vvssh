package plugin

import (
	"strings"
	"errors"
	"os"
)

/*

Plugins命令：
var_env [本地变量名称] [环境变量名称]               将环境变量存入本地变量，不会进行检查，运行时没有环境变量将报错
 */

func VarEnvRegisterPlugin()RemoteSSHCmdPlugin{
	res:=RemoteSSHCmdPlugin{
		Name:"var_env",
		Check:varEnvCheck,
		Run:varEnvRun,
		Unload:nil,
		Context:&SSHContext{
			Vars:nil,
			SSHClient:nil,
		},
	}
	return res
}



func varEnvCheck(context *SSHContext,commands []string)error{
	if len(commands)!=2 {
		return errors.New("arg must like this $[local var] [env var]")
	}

	if !strings.HasPrefix(commands[0],"$[")||
		!strings.HasSuffix(commands[0],"]"){
		return errors.New("line var format error! must like $[var_name]")
	}

	if _,find:=os.LookupEnv(commands[1]);!find{
		return errors.New(" can't find env var , name: \""+commands[1]+"\"")
	}


	return nil
}


func varEnvRun(context *SSHContext,commands []string)error {
	if envVar,find:=os.LookupEnv(commands[1]);find{
		varName:=strings.TrimLeft(commands[0],"$[")
		varName=strings.TrimRight(varName,"]")
		(*context.Vars)[varName]=envVar
	}else{
		return errors.New(" can't find env var , name: \""+commands[1]+"\"")
	}
	return nil
}