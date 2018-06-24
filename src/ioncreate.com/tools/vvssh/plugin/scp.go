package plugin

import (
	"errors"
	"os"
	"path/filepath"
	"fmt"
	"io"
	"bufio"
	"io/ioutil"
	"strings"
	"strconv"
)

/*

Plugins命令：
scp 	[本地文件名/或目录名] [远程文件路径] 		执行SCP传送任务
 */
func ScpCmdRegisterPlugin()RemoteSSHCmdPlugin{
	res:=RemoteSSHCmdPlugin{
		Name:"scp",
		Check:scpCheck,
		Run:scpRun,
		Unload:nil,
		Context:&SSHContext{
			Vars:nil,
			SSHClient:nil,
			//CommandChan:nil,
		},
	}
	return res
}


func scpCheck(context *SSHContext,commands []string)error{

	if len(commands)!=2 {
		return errors.New("bad args! should be:#==scp [local file/path] [remote path]")
	}

	localFile:=commands[0]

	if !HasVars(context,localFile){
		if _, err := os.Stat(localFile); os.IsNotExist(err) {
			return errors.New("bad args! Local File/Path is not exist")
		}
		if !filepath.IsAbs(commands[0]){
			real,err:=filepath.Abs(commands[0])
			if err!=nil{
				return err
			}
			println("Warn=>Real Path:",real)
		}
	}



	if (!filepath.IsAbs(commands[1]))&&
		(!HasVars(context,commands[1])){
		return errors.New("\""+commands[1]+"\" is bad args! Remote Path is not absolute path or has var")
	}

	return nil
}

func scpRun(context *SSHContext,command []string)error{



	println("start SCP..")

	cmdStr:=strings.Join(command," ")
	cmdStr=context.SetLineVars(cmdStr)
	allCmd :=strings.Split(cmdStr," ")
	if context==nil||context.SSHClient==nil{
		return errors.New("empty context or ssh client")
	}

	println("start SCP.. from: ",allCmd[0]," to: ",strconv.Quote(allCmd[1]))
	fi, err := os.Stat(allCmd[0])

	if err != nil {
		return err
	}

	session,err:=context.SSHClient.NewSession()
	if err != nil {
		return err
	}
	SessionIOIn, err := session.StdinPipe()
	if err != nil {
		return err
	}
	SessionIOOut,err:=session.StdoutPipe()
	if err != nil {
		return err
	}

	ResultOutReader:=bufio.NewReader(SessionIOOut)

	//var DoneChan chan int

	if err := session.Start(fmt.Sprintf("scp -rt %s", allCmd[1])); err != nil {
		return err
	} else {
		err=scpProtocolReadResult(ResultOutReader)
		if err!=nil{
			return err
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():
			err=scpDir(SessionIOIn,ResultOutReader, allCmd[0])
			if err!=nil{
				return err
			}
		case mode.IsRegular():
			println("start SCP..File")
			err=scpFile(SessionIOIn,ResultOutReader, allCmd[0])
			if err!=nil{
				return err
			}
		}


		println("scp done!")
		SessionIOIn.Close()
		session.Close()
	}
	return nil
}

func scpDir(SessionIOIn io.Writer,ResultOutReader *bufio.Reader,srcDir string)( error){
    srcDirInfo, err := os.Stat(srcDir)
	if err != nil {
		return  err
	}

	_, err = fmt.Fprintf(SessionIOIn, "%c%#4o %d %s\n", 'D', srcDirInfo.Mode()&os.ModePerm, 0, filepath.Base(srcDir))
	if err != nil {

		return fmt.Errorf("failed to write scp start directory header: err=%s", err)
	}
	err=scpProtocolReadResult(ResultOutReader)
	if err != nil {

		return fmt.Errorf("failed to write scp start directory header: err=%s", err)
	}


	DirFileInfos, err := ioutil.ReadDir(srcDir)

	for idx := range DirFileInfos {
		currentFileI:=DirFileInfos[idx]
		curFullFileName:=filepath.Join(srcDir,currentFileI.Name())
		switch mode := currentFileI.Mode(); {
		case mode.IsDir():
			err:=scpDir(SessionIOIn,ResultOutReader,curFullFileName)
			if err!=nil{
				return err
			}
		case mode.IsRegular():
			err:=scpFile(SessionIOIn,ResultOutReader,curFullFileName)
			if err!=nil{
				return err
			}
		}
	}
	_, err = fmt.Fprintf(SessionIOIn, "%c\n", 'E')
	if err != nil {
		return fmt.Errorf("failed to write scp end directory header: err=%s", err)
	}
	return scpProtocolReadResult(ResultOutReader)
}

func scpProtocolReadResult(ResultOutReader *bufio.Reader) error{
	b, err := ResultOutReader.ReadByte()
	if err != nil {
		return fmt.Errorf("failed to read scp reply type: err=%s", err)
	}
	if b == '\x00' {
		return nil
	}
	if (b == '\x01') || (b == '\x02') {
		line, err := ResultOutReader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read scp reply message: err=%s", err)
		}
		return fmt.Errorf("fatal Error %s",line)
	}else {
			return fmt.Errorf("unexpected scp reply type: %v", b)
	}

}

func scpFile(SessionIOIn io.Writer,ResultOutReader *bufio.Reader,srcFile string)( error){
	println("start SCP..File Copy")

	srcFileInfo, err := os.Stat(srcFile)
	if err != nil {
		return  err
	}
	File, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer File.Close()

	println("start SCP..File Copy ing..."+srcFileInfo.Name()," ",srcFileInfo.Size()," ",strconv.Itoa(int(srcFileInfo.Mode())))

	_, err = fmt.Fprintf(SessionIOIn, "%c%#4o %d %s\n", 'C',srcFileInfo.Mode()&os.ModePerm, srcFileInfo.Size(), srcFileInfo.Name())

	if err!=nil{
		println("start SCP..File Copy in... Error:"+err.Error())
	}
	println("start SCP..File Copy Ready...")
	_,err=io.Copy(SessionIOIn, File)
	if err!=nil{
		println("start SCP..File Copy in... Error:"+err.Error())
	}

	err= scpProtocolReadResult(ResultOutReader)
	if err!=nil{
		println("start SCP..File Copy Result... Error 1:"+err.Error())
	}

	println("start SCP..File Copy Done...")
	_,err=fmt.Fprint(SessionIOIn, "\x00")
	if err!=nil{
		println("start SCP..File Copy in... Error:"+err.Error())
	}

	println("start SCP..File Copy end...")

	return scpProtocolReadResult(ResultOutReader)
}


func scpFile2(SessionIOIn io.Writer,ResultOutReader *bufio.Reader,srcFile string)(chan int, error){

	println("start SCP..File Copy")
	DoneChan:=make(chan int)
	srcFileInfo, err := os.Stat(srcFile)
	if err != nil {
		return nil, err
	}

	go func() {
		defer func() {DoneChan<-1}()
		File, err := os.Open(srcFile)
		if err != nil {
			return
		}
		defer File.Close()
		println("start SCP..File Copy ing..."+srcFileInfo.Name()," ",srcFileInfo.Size()," ",strconv.Itoa(int(srcFileInfo.Mode())))

		_, err = fmt.Fprintf(SessionIOIn, "%c%#4o %d %s\n", 'C',srcFileInfo.Mode()&os.ModePerm, srcFileInfo.Size(), srcFileInfo.Name())

		if err!=nil{
			println("start SCP..File Copy in... Error:"+err.Error())
		}
		println("start SCP..File Copy Ready...")
		_,err=io.Copy(SessionIOIn, File)
		if err!=nil{
			println("start SCP..File Copy in... Error:"+err.Error())
		}

		err= scpProtocolReadResult(ResultOutReader)
		if err!=nil{
			println("start SCP..File Copy Result... Error 1:"+err.Error())
		}

		println("start SCP..File Copy Done...")
		_,err=fmt.Fprint(SessionIOIn, "\x00")
		if err!=nil{
			println("start SCP..File Copy in... Error:"+err.Error())
		}

		err= scpProtocolReadResult(ResultOutReader)
		if err!=nil{
			println("start SCP..File Copy Result... Error 2:"+err.Error())
		}


		println("start SCP..File Copy end...")


	}()

	println("start SCP..File Copy main...end")
	return DoneChan,nil
}
