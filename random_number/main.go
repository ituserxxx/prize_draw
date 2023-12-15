package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	qrcode "github.com/skip2/go-qrcode"
	"gopkg.in/antage/eventsource.v1"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

var cache sync.Map

var sseHandler eventsource.EventSource

func main() {
	// 开启一个 sse
	sseHandler = eventsource.New(nil, nil)
	defer sseHandler.Close()

	fmt.Println("http://ip:8002")
	// 启动HTTP服务器
	http.ListenAndServe(":8002", RegisterRoute())
}

// cache Demo
func cacheDemo() {
	cache.Store("key", "value")
	// 获取缓存
	value, found := cache.Load("key")
	if found {
		fmt.Println(value)
	}
	// 删除缓存
	cache.Delete("key")
}

// qrcode demo
func qrcodeDemo() {
	data := "http://www.cj.yirisanqiu.com/" // 自定义参数的 URL 或文本
	filePath := "index_qrcode.png"          // 存放目录和文件名
	// 生成二维码
	err := qrcode.WriteFile(data, qrcode.Medium, 256, filePath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("二维码生成成功！")
}

func SendSee(id, v string) {
	sseHandler.SendEventMessage(v, id, id)
}

// echo "OPENAI_API_KEY=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" > .env
func RegisterRoute() http.Handler {
	router := http.NewServeMux()
	//建立路由规则，将所有请求交给静态文件处理器处理

	router.Handle("/", http.FileServer(http.Dir("web")))
	router.Handle("/mobile", http.FileServer(http.Dir("web")))
	router.Handle("/cj_qrcode/", http.StripPrefix("/cj_qrcode/", http.FileServer(http.Dir("./cj_qrcode"))))
	router.Handle("/events", sseHandler)

	// POST请求处理
	router.HandleFunc("/hello", middleware(apiHandlerTest))                           // 测试
	router.HandleFunc("/api/get_cj_info", middleware(Crontroller.HandlerGetCjInfo))   // 获取抽奖信息
	router.HandleFunc("/api/save_cj_info", middleware(Crontroller.HandlerSaveCjInfo)) // 保存抽奖信息
	router.HandleFunc("/api/save_gs_name", middleware(Crontroller.HandlerSaveGsName)) // 保存公司名称
	router.HandleFunc("/api/user_view_cj", middleware(Crontroller.HandlerUserViewCj)) // 用户查看抽奖
	router.HandleFunc("/api/user_join_cj", middleware(Crontroller.HandlerUserJoinCj)) // 用户参与抽奖

	router.HandleFunc("/api/save_zj_user", middleware(Crontroller.HandlerSaveZjUser)) // 保存中奖用户

	router.HandleFunc("/api/send_cj_dyamic_msg", middleware(Crontroller.HandlerSendCjDyamicMsg)) // 发送抽奖互动消息

	return router
}

var Crontroller = &uLogic{}

type uLogic struct {
}

type cjOption struct {
	Name   string   `json:"name"`    // 奖品名称	Level int    `json:"level"` // 几等奖
	Num    int      `json:"num"`     //奖品数量
	ZjList []string `json:"zj_list"` //中奖人编号列表
}
type JoinUser struct {
	UUID     string `json:"uuid"`
	CjNumber int    `json:"cj_number"`
}
type cj struct {
	CJId         string     `json:"cj_id"` // uuid当做cjid
	GsName       string     `json:"gs_name"`
	CjQrcode     string     `json:"cj_qrcode"`
	L            []cjOption `json:"l"`
	JoinUserList []JoinUser `json:"join_user_list"` // 参与用户列表
	PersonTotal  int        `json:"person_total"`   //当前参与人数
	Lock         sync.RWMutex
}

var cjManager = map[string]*cj{}
var cjMLock sync.RWMutex

type GetCjInfoResp struct {
	GsName       string     `json:"gs_name"`
	CjQrcode     string     `json:"cj_qrcode"`
	L            []cjOption `json:"l"`
	PersonTotal  int        `json:"person_total"` //当前参与人数
	JoinUserList []JoinUser `json:"join_user_list"`
}

// HandlerGetCjInfo 发起人才能进的抽奖信息页面
func (uc *uLogic) HandlerGetCjInfo(w http.ResponseWriter, r *http.Request) {
	var res = &GetCjInfoResp{
		L:            make([]cjOption, 0),
		JoinUserList: make([]JoinUser, 0),
	}
	uuid := r.Header.Get("uuid")
	v, ok := cjManager[uuid]
	// 当前抽奖不存在
	if ok == false { // 活动发起人的 uuid == lid  抽奖id
		var info = &cj{
			L:            make([]cjOption, 0),
			JoinUserList: make([]JoinUser, 0),
		}
		info.CJId = uuid
		info.CjQrcode = getQrcodeUrl(uuid)
		cjMLock.Lock()
		defer cjMLock.Unlock()
		cjManager[uuid] = info
		res.CjQrcode = info.CjQrcode
	}
	// 当前抽奖存在
	if ok == true {
		res.CjQrcode = v.CjQrcode
		res.PersonTotal = v.PersonTotal
		res.L = v.L
		res.GsName = v.GsName
		res.JoinUserList = v.JoinUserList
	}
	respOk(w, res)
}

type SaveCjInfoReq struct {
	L []cjOption `json:"l"`
}

func (uc *uLogic) HandlerSaveCjInfo(w http.ResponseWriter, r *http.Request) {
	// 解析请求体数据到结构体
	var list *SaveCjInfoReq
	err := json.NewDecoder(r.Body).Decode(&list)
	if err != nil {
		respErr(w, "json unmarshal err "+err.Error())
		return
	}
	uuid := r.Header.Get("uuid")
	v, ok := cjManager[uuid]
	if ok == false {
		respErr(w, "抽奖不存在~~")
		return
	}
	if ok == true {
		v.Lock.RLock()
		defer v.Lock.RUnlock()
		v.L = list.L
	}
	respOk(w, "")
}

type saveGsNameReq struct {
	GsName string `json:"gs_name"`
}

func (uc *uLogic) HandlerSaveGsName(w http.ResponseWriter, r *http.Request) {
	var req saveGsNameReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respErr(w, "json unmarshal err "+err.Error())
		return
	}
	req.GsName = strings.TrimSpace(req.GsName)
	if len(req.GsName) == 0 {
		respErr(w, "活动名称不能为空")
		return
	}
	uuid := r.Header.Get("uuid")
	v, ok := cjManager[uuid]
	if ok == false {
		respErr(w, "抽奖不存在~~")
		return
	}
	if ok == true {
		v.Lock.RLock()
		defer v.Lock.RUnlock()
		v.GsName = req.GsName
	}
	respOk(w, "")
}

type JoinCjReq struct {
	LID string `json:"lid"`
}

type UserViewCjResp struct {
	Num    int        `json:"num"`
	GsName string     `json:"gs_name"`
	L      []cjOption `json:"l"`
}

func (uc *uLogic) HandlerUserViewCj(w http.ResponseWriter, r *http.Request) {
	var req JoinCjReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respErr(w, "json unmarshal err "+err.Error())
		return
	}

	req.LID = strings.TrimSpace(req.LID)
	if len(req.LID) == 0 {
		respErr(w, "活动id不能为空")
		return
	}
	uuid := r.Header.Get("uuid")
	v, ok := cjManager[req.LID]
	if ok == false {
		respErr(w, "抽奖不存在~~")
		return
	}
	if ok == true {
		var userNum int
		for _, u := range v.JoinUserList {
			if u.UUID == uuid {
				userNum = u.CjNumber
				break
			}
		}
		respOk(w, UserViewCjResp{
			Num:    userNum,
			GsName: v.GsName,
			L:      v.L,
		})
	}
}

type SaveZjUserReq struct {
	LID  string `json:"lid"`
	Leve int    `json:"leve"`
}

func (uc *uLogic) HandlerSaveZjUser(w http.ResponseWriter, r *http.Request) {
	var req SaveZjUserReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respErr(w, "json unmarshal err "+err.Error())
		return
	}

	zjUser := r.Header.Get("zj_str")
	if strings.TrimSpace(zjUser) == "" {
		http.Error(w, " Not Allowed Connect ", http.StatusBadRequest)
		return
	}
	req.LID = strings.TrimSpace(req.LID)
	if len(req.LID) == 0 {
		respErr(w, "活动id不能为空")
		return
	}
	if req.Leve == 0 {
		respErr(w, "leve err")
		return
	}
	if req.LID != r.Header.Get("uuid") {
		http.Error(w, " Not Allowed Connect ", http.StatusBadRequest)
		return
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(zjUser)
	if err != nil {
		http.Error(w, " Not Allowed Connect ", http.StatusBadRequest)
		return
	}

	decodedStr := string(decodedBytes)
	zjidL := strings.Split(decodedStr, ",")

	v, ok := cjManager[req.LID]
	if ok == false {
		respErr(w, "抽奖不存在~~")
		return
	}
	if ok == true {
		v.Lock.RLock()
		defer v.Lock.RUnlock()
		for i, _ := range v.L {
			if i+1 == req.Leve {
				v.L[i].ZjList = zjidL
				break
			}
		}
		respOk(w, "ok")
	}
}
func (uc *uLogic) HandlerUserJoinCj(w http.ResponseWriter, r *http.Request) {
	var req JoinCjReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respErr(w, "json unmarshal err "+err.Error())
		return
	}

	req.LID = strings.TrimSpace(req.LID)
	if len(req.LID) == 0 {
		respErr(w, "活动id不能为空")
		return
	}
	uuid := r.Header.Get("uuid")

	v, ok := cjManager[req.LID]
	if ok == false {
		respErr(w, "抽奖不存在~~")
		return
	}
	if ok == true {
		v.Lock.RLock()
		defer v.Lock.RUnlock()
		var isJoin bool
		var userNum int
		for _, u := range v.JoinUserList {
			if u.UUID == uuid {
				isJoin = true
				userNum = u.CjNumber
				break
			}
		}
		if isJoin == false {
			v.PersonTotal += 1
			v.JoinUserList = append(v.JoinUserList, JoinUser{
				UUID:     uuid,
				CjNumber: v.PersonTotal,
			})
			userNum = v.PersonTotal
			SendSee(v.CJId, fmt.Sprintf("join-用户%s刚刚加入", uuid))
			SendSee(v.CJId, fmt.Sprintf("num-%d", userNum))
		}
		respOk(w, userNum)
	}
}

type SendCjDyamicMsgReq struct {
	LID string `json:"lid"`
	Msg string `json:"msg"`
}

func (uc *uLogic) HandlerSendCjDyamicMsg(w http.ResponseWriter, r *http.Request) {
	var req SendCjDyamicMsgReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respErr(w, "json unmarshal err "+err.Error())
		return
	}
	req.LID = strings.TrimSpace(req.LID)
	if len(req.LID) == 0 {
		respErr(w, "活动id不能为空")
		return
	}
	req.Msg = strings.TrimSpace(req.Msg)
	if len(req.Msg) == 0 {
		respErr(w, "")
		return
	}
	uuid := r.Header.Get("uuid")
	// 获取当当前用户的编号
	var uNumber int
	v, ok := cjManager[req.LID]
	if ok == false {
		respErr(w, "抽奖不存在~~")
		return
	}
	for _, u := range v.JoinUserList {
		if u.UUID == uuid {
			uNumber = u.CjNumber
			break
		}
	}
	if uNumber == 0 {
		respErr(w, "请先参与该活动")
		return
	}
	SendSee(v.CJId, fmt.Sprintf("msg-抽奖号%d说：%s", uNumber, req.Msg))
	respOk(w, "")
}

// 自定义中间件函数
func middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")                          // 设置允许跨域的域名列表
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")        // 设置允许跨域的请求方式
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,X-token,uuid") // 设置允许的请求头部
		// 对于预检请求（OPTIONS），直接返回成功
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.TrimSpace(r.Header.Get("uuid")) == "" {
			http.Error(w, " Not Allowed Connect ", http.StatusBadRequest)
			return
		}
		// 调用下一个处理函数
		next(w, r)
	}
}

func apiHandlerTest(w http.ResponseWriter, r *http.Request) {
	// 获取路由参数
	name := r.URL.Query().Get("name")
	respOk(w, name)
	respErr(w, name)
}

type resp struct {
	Data interface{} `json:"data"`
	Code int         `json:"code"`
}

func respHandle(w http.ResponseWriter, code int, data interface{}) ([]byte, error) {
	response := resp{
		Data: data,
		Code: code,
	}
	w.Header().Set("Content-Type", "application/json") // 设置响应头的Content-Type为"application/json"
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	return jsonResponse, nil
}
func respOk(w http.ResponseWriter, data interface{}) {
	jsonResponse, err := respHandle(w, 33, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(jsonResponse))

}
func respErr(w http.ResponseWriter, data string) {
	jsonResponse, err := respHandle(w, 44, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprint(w, string(jsonResponse))
}

/*---------qrcode--------------*/
func createQrcodeUrl(uuid string) error {
	fileName := fmt.Sprintf("cj_qrcode/%s.png", uuid)

	// 生成二维码
	err := qrcode.WriteFile("http://www.cj.yirisanqiu.com/mobile?lid="+uuid, qrcode.Medium, 256, fileName)
	if err != nil {
		return err
	}
	return nil
}
func getQrcodeUrl(uuid string) string {
	k := "cj_qrcode-" + uuid
	value, found := cache.Load(k)
	if found {
		return fmt.Sprintf("%v", value)
	}
	err := createQrcodeUrl(uuid)
	if err != nil {
		fmt.Println("createQrcodeUrl", err.Error())
		return ""
	}
	cache.Store(k, uuid)
	return fmt.Sprintf("cj_qrcode/%s.png", uuid)
}
func delQrcodeUrl(uuid string) {
	k := "cj_qrcode-" + uuid
	cache.Delete(k)
	os.Remove(fmt.Sprintf("cj_qrcode/%s.png", uuid))
}

//
///* ---------ws------------*/
//type ClientManager struct {
//	Clients    map[string]*WsClient
//	Broadcast  chan []byte
//	Register   chan *WsClient
//	Unregister chan *WsClient
//}
//
//type WsClient struct {
//	Socket *websocket.Conn
//	Send   chan []byte
//	UUID   string
//}
//
//var Manager = ClientManager{
//	Register:   make(chan *WsClient),
//	Unregister: make(chan *WsClient),
//	Clients:    make(map[string]*WsClient),
//}
//
//var upgrader = websocket.Upgrader{
//	CheckOrigin: func(r *http.Request) bool {
//		return true
//	},
//}
//
//func wsHandler(w http.ResponseWriter, r *http.Request) {
//	conn, err := upgrader.Upgrade(w, r, nil)
//	if err != nil {
//		log.Println("Upgrade:", err)
//		return
//	}
//	defer conn.Close()
//	uuid := r.Header.Get("uuid")
//	//可以添加用户信息验证
//	client := &WsClient{
//		Socket: conn,
//		Send:   make(chan []byte),
//		UUID:   uuid,
//	}
//
//	Manager.Register <- client
//	go client.Read()
//	go client.Write()
//}
//
//func (c *WsClient) Read() {
//	defer func() {
//		Manager.Unregister <- c
//		_ = c.Socket.Close()
//	}()
//	for {
//		c.Socket.PongHandler()
//		_, message, err := c.Socket.ReadMessage()
//		if err != nil {
//			_ = c.Socket.Close()
//			break
//		}
//		fmt.Println(string(message))
//		if len(message) == 0 {
//			c.Send <- message
//			continue
//		}
//		fmt.Println(string(message))
//	}
//}
//
//func (c *WsClient) Write() {
//	defer func() {
//		Manager.Unregister <- c
//		_ = c.Socket.Close()
//	}()
//	for {
//		select {
//		case message, ok := <-c.Send:
//			if !ok {
//				_ = c.Socket.WriteMessage(websocket.CloseMessage, []byte{})
//				return
//			}
//			_ = c.Socket.WriteMessage(websocket.TextMessage, message)
//		}
//	}
//}
//
//// WSStart is  项目运行前, 协程开启start -> go Manager.Start()
//func (manager *ClientManager) WSStart() {
//	for {
//		select {
//		case conn := <-Manager.Register:
//			Manager.Clients[conn.UUID] = conn
//		case conn := <-Manager.Unregister:
//			if _, ok := Manager.Clients[conn.UUID]; ok {
//				close(conn.Send)
//				delete(Manager.Clients, conn.UUID)
//			}
//		}
//	}
//}
//
//func SendToWeb(uuid string, data interface{}) {
//	for _, c := range Manager.Clients {
//		if c.UUID == uuid {
//			b, _ := json.Marshal(data)
//			c.Send <- b
//		}
//	}
//}
