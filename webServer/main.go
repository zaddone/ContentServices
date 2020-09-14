package main
import(
	"fmt"
	"time"
	"github.com/gin-gonic/gin"
	"flag"
	"net/http"
	"encoding/json"
	"strings"
	//"io/ioutil"
	"ContentServices/content"
)
var (
	Router  = gin.Default()
	port = flag.String("p","8080","port")
)
func init(){
	flag.Parse()
	Router.Static("/static","./static")
	Router.LoadHTMLGlob("./templates/*")

	Router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK,gin.H{"msg":"success"})
	})
	Router.POST("/update", func(c *gin.Context) {
		var db interface{}
		err := json.NewDecoder(c.Request.Body).Decode(&db)
		c.Request.Body.Close()
		if err != nil {
			c.JSON(http.StatusNotFound,err)
			return
		}
		con := NewContent(db)
		err = con.UpdateInfo()
		if err != nil {
			c.JSON(http.StatusNotFound,err)
			return
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

	mdb := map[string]interface{}{}
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
		c.SetWord(mdb["words"].([]string))
	}else{
		c.SetWords()
	}
	c.SetId(strings.Join(c.GetWords(),""))
	return c

}

func main(){
	Router.Run(":"+*port)

}
