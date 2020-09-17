package main
import(
	"fmt"
	"io"
	"time"
	"github.com/gin-gonic/gin"
	"flag"
	"net/http"
	"encoding/json"
	"strings"
	//"io/ioutil"
	"ContentServices/content"
	"ContentServices/wxmsgb"
	"sync"
	//"os"
)
var (
	Router  = gin.Default()
	port = flag.String("p","8080","port")
	//writerChan = make(chan *content.Content,100)
	//TmpUpdateForWX = "tmpWX.db"
	appid = flag.String("appid","wx92ebd09c7b0d944f","appid")
	sec = flag.String("sec","b3005d3c298e27b60ee1f90d188a9d86","sec")
	env = flag.String("env","guomi-2i7wu","env")
)
func uploadWX() (err error){
	var wi sync.WaitGroup
	err = content.ReadTmpAll(
		func(c io.Reader){
			wi.Add(1)
			go func(){
				err := wxmsgb.UploadWX("content",c)
				if err != nil {
					fmt.Println(err)
				}
				wi.Done()
			}()
		},
		func(w io.Reader){
			wi.Add(1)
			go func(){
				err := wxmsgb.UploadWX("keywords",w)
				if err != nil {
					fmt.Println(err)
				}
				wi.Done()
			}()
		},
	)
	wi.Wait()
	return content.ClearTmpDB()

}

func init(){
	flag.Parse()
	wxmsgb.Reload(*appid,*sec,*env)
	//Router.Static("/static","./static")
	//Router.LoadHTMLGlob("./templates/*")

	Router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK,gin.H{"msg":"success"})
	})
	Router.GET("/syncwx", func(c *gin.Context) {
		c.JSON(http.StatusOK,gin.H{"msg":uploadWX()})
	})
	Router.POST("/update", func(c *gin.Context) {
		var db map[string]interface{}
		err := json.NewDecoder(c.Request.Body).Decode(&db)
		c.Request.Body.Close()
		if err != nil {
			//panic(err)
			fmt.Println(err)
			c.JSON(http.StatusNotFound,err)
			return
		}
		//fmt.Println(db)
		con := NewContent(db)
		//fmt.Println(con)
		err = con.UpdateInfo()
		if err != nil {
			c.JSON(http.StatusNotFound,err)
			return
		}
		err = con.UpdateTmp()
		if err != nil {
			fmt.Println(err)
		}
		c.JSON(http.StatusOK,con)
		return
	})
	Router.GET("/search", func(c *gin.Context) {
		var li []interface{}
		err := content.SearchWithWords(c.Query("q"),20,func(o interface{}){
			li = append(li,o)
		})
		if err != nil {
			fmt.Println(err)
			c.JSON(http.StatusNotFound,err)
			return
		}
		c.JSON(http.StatusOK,li)
	})

}

func NewContent(db interface{}) (c *content.Content) {
	//fmt.Println(db)
	mdb := db.(map[string]interface{})
	c = &content.Content{
		Title:mdb["Title"].(string),
		Content:mdb["Content"].(string),
		Author:mdb["Author"].(string),
		Site:mdb["Site"].(string),
		Type:int(mdb["Type"].(float64)),
		Update:time.Now().Unix(),
		//words:mdb["words"].([]string),
	}
	if mdb["words"] != nil {
		words := mdb["words"].([]interface{})
		ws:=make([]string,0,len(words))
		for _,w := range words {
			ws = append(ws,w.(string))
		}
		c.SetWord(ws)
	}else{
		c.SetWords()
	}
	c.SetId(strings.Join(c.GetWords(),""))
	return c

}

func main(){
	Router.Run(":"+*port)

}
