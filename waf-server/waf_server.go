package main

import (
	"flag"
	"github.com/BurntSushi/toml"
	"github.com/kumustone/panda-waf"
	"github.com/kumustone/tcpstream"
	"github.com/natefinch/lumberjack"
	"log"
)

const WafMsgVersion uint8 = 1

type Server struct {
	WafServerAddress string
	HttpAPIAddress   string
}

type WafServerConf struct {
	Server Server
}

var (
	confFile = flag.String("c", "./waf_server.conf", "Config file")
	logPath  = flag.String("l", "./log", " log path")
	rulePath = flag.String("r", "./rules", " rule path")
)

func main() {
	flag.Parse()

	c := WafServerConf{}

	if _, err := toml.DecodeFile(*confFile, &c); err != nil {
		log.Fatal("Can not decode config file ", err.Error())
		return
	}

	defer panda_waf.PanicRecovery(true)
	log.SetOutput(&lumberjack.Logger{
		Filename:   "waf_server.log",
		MaxSize:    10,
		MaxBackups: 10,
		MaxAge:     30,
	})

	if err := panda_waf.InitRulePath(*rulePath); err != nil {
		log.Fatal("InitRulePath : ", err.Error())
	}

	log.Println("waf-server listen at : ", c.Server.WafServerAddress)
	if err := tcpstream.NewTCPServer(c.Server.WafServerAddress, &ServerHandler{}).Serve(); err != nil {
		log.Println("server : ", err.Error())
	}

	select {}
}

type ServerHandler struct{}

func (*ServerHandler) OnData(conn *tcpstream.TcpStream, msg *tcpstream.Message) error {
	request := &panda_waf.WafHttpRequest{}
	if err := request.UnmarshalJSON(msg.Body); err != nil {
		return err
	}

	var resp *panda_waf.WafProxyResp
	for _, c := range panda_waf.CheckList {
		resp = c.CheckRequest(request)
		if resp.RuleName != "" {
			break
		}
	}

	body, _ := resp.MarshalJSON()
	respMsg := tcpstream.Message{
		Header: tcpstream.ProtoHeader{
			Seq: msg.Header.Seq,
		},
		Body: body,
	}

	return conn.Write(&respMsg)
}

func (*ServerHandler) OnConn(conn *tcpstream.TcpStream) {

}

func (*ServerHandler) OnDisConn(conn *tcpstream.TcpStream) {

}
