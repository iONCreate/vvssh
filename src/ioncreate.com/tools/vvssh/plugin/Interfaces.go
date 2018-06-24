package plugin

import (
	"golang.org/x/crypto/ssh"
	"bufio"
	"io"
	"regexp"
)

/*
若不需要使用相应功能的，直接将功能函数设置成nil，
所有Plugins必须实现CheckFunc

私有Plugin设计

所有都放到当前目录下Plugins目录

载入时首先搜索，并运行Plugin中的 VSSHCmdRegisterPlugin()， 等于下面 RegisterPluginFunc 函数原型。 来完成注册，剩下等待回调

 */
type RemoteSSHCmdPlugin struct{
	Name string
	//Register RegisterPluginsFunc
	Check   CheckFunc
	Run     RunFunc
	Unload  UnloadPluginFunc
	Context *SSHContext
}

type RegisterPluginFunc func()RemoteSSHCmdPlugin
type CheckFunc func(context *SSHContext,commands []string)error //ssh *ssh.SSHService,in io.WriteCloser, out io.Reader,
type RunFunc func(context *SSHContext,command []string)error    //ssh *ssh.SSHService,in io.WriteCloser, out io.Reader,
type UnloadPluginFunc func() error
type SSHOutputLineHookFunc func(line string)



type SSHContext struct {
	VarRegex *regexp.Regexp
	Vars *map[string]string
	SSHClient *ssh.Client
	SSHConsoleReader *bufio.Reader
	SSHConsoleWriter io.Writer
	SSHConsoleReadToPromote func(ioReader *bufio.Reader,hook SSHOutputLineHookFunc)error
	SSHConsoleWriteCommand func(ioReader *bufio.Reader,cmd string) error
	SetLineVars func(string)string
	//CommandChan chan <-string
}

func HasVars(c *SSHContext,s string)bool{
	if c.VarRegex==nil{
		return false
	}
	str1:=c.VarRegex.FindString(s)
	if str1!="" {
		return true
	}
	return false
}