package utils

import (
	"golang.org/x/crypto/ssh"
	"os"
	"os/user"
	"io/ioutil"
	"io"
	"fmt"
	"bufio"
	"net"
	"golang.org/x/crypto/ssh/agent"
	"strings"
	"ioncreate.com/tools/vvssh/plugin"
	"time"
	"sync"
)

type SSHSessionIO struct {
	In io.WriteCloser
	Out io.Reader
	Err io.Reader
}


type SSHService struct {
	User     string
	Server   string
	Key      string
	Password string
	TimeOut int
	IsAutoSudo bool

	Client           *ssh.Client
	ShellPromoteByte byte
	RemoteMainIO     *SSHSessionIO


	MainSession *ssh.Session

	feedDogChan chan<-int
	stopDogChan chan<-int
	dogWaitGroup *sync.WaitGroup

}

func NewSSHService()*SSHService {
	res := &SSHService{
		RemoteMainIO: &SSHSessionIO{},
		//Control:   &SSHControlIO{
		//	CommandChan:nil,
		//	CommandClosed:Common.NewAtomBool(),
		//},
		IsAutoSudo:false,
		Key:"/.ssh/id_rsa",
		//MainSessionCloseChan:make(chan struct{}),
	}
	return res
}

func CheckError(err error, msg string) {
	if err != nil {
		fmt.Print(`
 ===============RUN ERROR=======================
`)
		fmt.Printf("%s \nDetail:\n%v\n\n", msg, err)
		//PrintUsage()
		fmt.Print("Please use -h or --help flag to check usages!\n")
		os.Exit(1)
	}

}

// connects to remote server using SSHService struct and returns *ssh.Session
func (ss *SSHService) Connect() error {
	// auths holds the detected ssh auth methods
	var auths []ssh.AuthMethod

	// figure out what auths are requested, what is supported
	if ss.Password != "" {
		auths = append(auths, ssh.Password(ss.Password))
	}

	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers))
		defer sshAgent.Close()
	}

	if pubkey, err := getKeyFile(ss.Key); err == nil {
		auths = append(auths, ssh.PublicKeys(pubkey))
	}

	config := &ssh.ClientConfig{
		User:            ss.User,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		BannerCallback:ssh.BannerDisplayStderr(),
	}

	// TODO:there should has StrictMode for sec
	//if ss.StrictMode{
	//	config.HostKeyCallback= func(hostname string, remote net.Addr, key ssh.PublicKey) error {
	//
	//	}
	//}

	var err error
	ss.Client,err= ssh.Dial("tcp", ss.Server, config)
	if err != nil {
		return  err
	}



	return nil
}

func (ss *SSHService)NewSession()(*ssh.Session,error){
	session, err := ss.Client.NewSession()
	return session,err
}



/*
有以下几种情况：
1. exit Remote 对端主动关闭 反应是 RemoteMainIO.IO 读写会遇到EOF或其他错误
2. 网络错误 本地被动关闭 反应是 RemoteMainIO.IO 读写会遇到EOF或其他错误
3. 主动退出包含异常 本地主动关闭

设置一个优雅退出的Chan， 专用go 管理，通道产生信息，执行下列退出工作：
1. 停止RunCommand并等待


实际上该程序仅有两个状态，
1。命令中 从发送命令后开始
2。命令完成 从读取标示符开始

真的太蠢，小题大做，直接序列进行即可，笨蛋！

 */


func (ss *SSHService)timeOutWatchDog(TimeOutSec int,TimeOutCallback func())(FeedDog chan<-int,StopDog chan<-int,DogWaitGroup *sync.WaitGroup){

	feedDog:=make(chan int,1000)
	stopDog:=make(chan int)
	dogWaitGroup:=&sync.WaitGroup{}
	dogWaitGroup.Add(1)
	go func(aFeedDog chan int,aStopDog chan int,aTimeOutSec int,aDogWaitGroup *sync.WaitGroup) {
		TimerOut:=time.Duration(aTimeOutSec)*time.Second
		timer:=time.NewTimer(TimerOut)
		ResetOutTimer:= func(timer *time.Timer,TimeOut time.Duration){
			if !timer.Stop() {
				select {
				case <-timer.C: //try to drain from the channel
				default:
				}
			}
			timer.Reset(TimeOut)
		}
		for {
			select {
			case <-aFeedDog:
				ResetOutTimer(timer,TimerOut)
			case <-timer.C:
				TimeOutCallback()
				ResetOutTimer(timer,TimerOut)
			case <-aStopDog:
				if !timer.Stop() {
					select {
					case <-timer.C: //try to drain from the channel
					default:
					}
				}
				println("Dog Stop!")
				aDogWaitGroup.Done()
				return
			}
		}
	}(feedDog,stopDog,TimeOutSec,dogWaitGroup)

	return feedDog,stopDog,dogWaitGroup
}

func (ss *SSHService) feedDog() {
	ss.feedDogChan<-1
}

 func (ss *SSHService) WriteCommandToSSHShell(IOReader *bufio.Reader,cmd string)error{
	 _,err:=ss.RemoteMainIO.In.Write([]byte(cmd + "\r\n"))
	 if err!=nil{
		 return err
	 }
	 for{
		 line, err := IOReader.ReadString('\n')
		 if err != nil {
			 os.Stdout.WriteString("\r\nError in SSH Read Thread Error:" + err.Error() + "\r\n")
			 break
		 }
		 ss.feedDog()

		 _, err = os.Stdout.WriteString(line)
		 if err != nil {
			 //??? What will case this
			 println("Write Std Err ")
		 }

		 lineCount := len(line)

		 b := []byte(line)
		 cmdStrCount :=len(cmd)
		 if (4+cmdStrCount)<=lineCount  {
			 c:=strings.TrimSpace(string(b[lineCount-2-cmdStrCount:]))
			 //println(">>",strconv.Quote(line),lineCount,cmdStrCount,string(b[lineCount-4-cmdStrCount-1]),"=>>",
				// string(b[lineCount-4-cmdStrCount]),
				// string(b[lineCount-3-cmdStrCount]),
				// "["+c+"]")
			 if b[lineCount-4-cmdStrCount] == ss.ShellPromoteByte &&
		 		b[lineCount-3-cmdStrCount] == ' '&&
		 		c==cmd { //assuming the $PS1 == 'sh-4.3$ '
				 //println("got one Result!")
				 break
			 }
		 }
	 }

	 return nil
 }

func (ss *SSHService) ReadStrToShellPromote(IOReader *bufio.Reader,ReadCallback plugin.SSHOutputLineHookFunc) error{
	sudoPromoteStr:="[sudo] password for "+ss.User+": \r\n"
	sudoLen:=len(sudoPromoteStr)
	for {
		line, err := IOReader.ReadString('\n')
		if err != nil {
			os.Stdout.WriteString("\r\nError in SSH Read Thread Error:" + err.Error() + "\r\n")
			break
		}
		ss.feedDog()

		lineCount := len(line)

		b := []byte(line)
		//_,err=os.Stdout.Write(buf[bufR:n])
		_, err = os.Stdout.WriteString(line)

		//println(strconv.Quote(line))
		if err != nil {
			//??? What will case this
			println("Write Std Err ")
		}
		if lineCount >= sudoLen &&
			ss.IsAutoSudo &&
			line == sudoPromoteStr {
			ss.RemoteMainIO.In.Write([]byte(ss.Password + "\r\n"))
			println("Write Password! ")
		}

		if lineCount > 4 {
			if b[lineCount-4] == ss.ShellPromoteByte && b[lineCount-3] == ' ' { //assuming the $PS1 == 'sh-4.3$ '
				//println("got one Promote!")
				break
			}
		}
		if ReadCallback!=nil{
			ReadCallback(line)
		}
	}

	return nil
}

func (ss *SSHService) EnableWatchDog(){
	ss.feedDogChan,ss.stopDogChan,ss.dogWaitGroup=ss.timeOutWatchDog(ss.TimeOut, func() {
		ss.RemoteMainIO.In.Write([]byte("\n"))
	})
}

func (ss *SSHService) StopWatchDog(){
	if ss.stopDogChan!=nil{
		close(ss.stopDogChan)
		ss.dogWaitGroup.Wait()
	}
}


func (ss *SSHService)WaitMainSession()error  {
	err:=ss.MainSession.Wait()
	return err
}

func(ss *SSHService) Stop()error{
	ss.StopWatchDog()
	err:=ss.MainSession.Close()
	return err
}






func (ss *SSHService) Start(){

	err:= ss.Connect()
	CheckError(err,"Connect Error!  Server: "+ss.Server)
	ss.MainSession,err= ss.NewSession()
	CheckError(err,"Open SSH Session Error!  Server: "+ss.Server)


	ss.RemoteMainIO.In, err = ss.MainSession.StdinPipe()
	CheckError(err,"Get Input Channel")
	ss.RemoteMainIO.Out, err = ss.MainSession.StdoutPipe()
	CheckError(err,"Get Out Channel")
	ss.RemoteMainIO.Err,err=ss.MainSession.StderrPipe()

	CheckError(err,"Get Err Channel")

	modes := ssh.TerminalModes{
		ssh.ECHO: 1, // enable echoing
		//ssh.IGNCR:         1, // Ignore CR on input.
		//ssh.ECHONL :1, // Echo NL even if ECHO is off.
		//ssh.IXOFF :1, //Enable input flow control.
		//ssh.IXON:1,//        Enable output flow control.
		//ssh.ICRNL:         1,     // Map CR to NL on input.
		//ssh.VSTATUS:       1,     //Prints system status line (load, command, pid, etc).
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	//ss.Control.CommandChan,ss.Control.CommandCloseWaitGroup=ss.MainSessionRun3()

	err = ss.MainSession.RequestPty("xterm", 388, 518, modes) //"xterm-256color"
	CheckError(err,"SSH RequestPty")
	os.Stdout.WriteString("Promote Byte Is:"+string(ss.ShellPromoteByte))
	os.Stdout.WriteString("\r\n>>>>>>>>>>>>SSH MAIN SESSION START!<<<<<<<<<<<<<<<<<<\r\n")

	err = ss.MainSession.Shell()
	CheckError(err, "start shell")
	ss.RemoteMainIO.In.Write([]byte("\r\n"))

	ss.EnableWatchDog()
	//ss.Control.CommandChan<-""
}





// returns ssh.Signer from user you running app home path + cutted key path.
// (ex. pubkey,err := getKeyFile("/.ssh/id_rsa") )
func getKeyFile(keypath string) (ssh.Signer, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	file := usr.HomeDir + keypath
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	pubkey, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}

	return pubkey, nil
}


