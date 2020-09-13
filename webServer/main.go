package main
import(
	"fmt"
	"github.com/gin-gonic/gin"
	"flag"
	"net/http"
	"io/ioutil"
)
var (
	Router  = gin.Default()
	port = flag.String("p","8080","port")
)
func init(){
	flag.Parse()
	//Router.Static("/static","./static")
	//Router.LoadHTMLGlob("./templates/*")

	Router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK,gin.H{"msg":"success"})
	})
	Router.POST("/update", func(c *gin.Context) {
		db, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(db))
	})
	Router.GET("/search", func(c *gin.Context) {
		var li []interface{}
		err := searchWithWords(c.Query("q"),20,func(o interface{}){
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
func main(){
	Router.Run(":"+*port)

}
