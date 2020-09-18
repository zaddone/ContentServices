package main
import(
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/boltdb/bolt"
	//"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
	"net/http"
	"io/ioutil"
	"regexp"
	"time"
	"encoding/binary"
	"flag"
	"strconv"
)
var(
	rl = regexp.MustCompile(`(\d{4}年\d{1,2}月\d{1,2}日)(.+)`)
	rll = regexp.MustCompile(`\S+`)
	dateStamp = "2006年01月02日"
	dateStamp_ = "2006年1月2日"
	dateStamp__ = "2006-1-2"
	calendarDB = "calendar.db"
	calendar = []byte("calendar")
	Router  = gin.Default()
	port = flag.String("p","8080","port")
)
func init(){
	flag.Parse()
	Router.GET("/info", func(c *gin.Context) {
		y,err := strconv.Atoi(c.Query("y"))
		if err != nil {
			c.JSON(http.StatusNotFound,err)
			return
		}
		m,err := strconv.Atoi(c.Query("m"))
		if err != nil {
			c.JSON(http.StatusNotFound,err)
			return
			//m = int(time.Now().Month())
		}
		d,err := strconv.Atoi(c.Query("d"))
		if err != nil {
			data,err := time.Parse(dateStamp__,fmt.Sprintf("%d-%d-1",y,m))
			if err != nil {
				c.JSON(http.StatusNotFound,err)
				return
			}
			mod := data.Month()
			err = openDB(calendarDB,false,func(t *bolt.Tx)error{
				b := t.Bucket(calendar)
				if b == nil {
					return fmt.Errorf("bucket is nil")
				}
				key := make([]byte,8)
				binary.BigEndian.PutUint64(key,uint64(data.Unix()))
				c_ := b.Cursor()
				var li  []string
				for k,v := c_.Seek(key);k!=nil;k,v = c_.Next(){
					md := int64(binary.BigEndian.Uint64(k))
					if mod != time.Unix(md,0).Month() {
						break
					}
					li = append(li,fmt.Sprintf("%d,%s",md,string(v)))
				}
				fmt.Println(li)
				c.JSON(http.StatusOK,gin.H{"msg":li})
				return nil
			})
			if err != nil {
				c.JSON(http.StatusNotFound,err)
			}
			return
		}
		data,err := time.Parse(dateStamp__,fmt.Sprintf("%d-%d-%d",y,m,d))
		if err != nil {
			c.JSON(http.StatusNotFound,err)
			return
		}
		err = openDB(calendarDB,false,func(t *bolt.Tx)error{
			b := t.Bucket(calendar)
			if b == nil {
				return fmt.Errorf("bucket is nil")
			}
			key := make([]byte,8)
			binary.BigEndian.PutUint64(key,uint64(data.Unix()))
			v := b.Get(key)
			if v == nil {
				return fmt.Errorf("%s is nil",data)
			}
			c.JSON(http.StatusOK,gin.H{"msg":fmt.Sprintf("%d,%s",data.Unix(),string(v))})
			return nil
		})
		if err != nil {
			c.JSON(http.StatusNotFound,err)
			return
		}
	})

}
func openDB(name string,writable bool,h func(*bolt.Tx)error)error{
	db,err := bolt.Open(name,0600,nil)
	if err != nil {
		return err
	}
	defer db.Close()
	t,err := db.Begin(writable)
	if err != nil {
		return err
	}
	if writable {
		defer t.Commit()
	}
	return h(t)
}
func DecodeBig5(s []byte) ([]byte, error) {
	I := bytes.NewReader(s)
	O := transform.NewReader(I, traditionalchinese.Big5.NewDecoder())
	d, e := ioutil.ReadAll(O)
	if e != nil {
		return nil, e
	}
	return d, nil
}
func downCalendar(n int,h func(interface{})error)error{
	res,err := http.Get(fmt.Sprintf("https://www.hko.gov.hk/tc/gts/time/calendar/text/files/T%dc.txt",n))
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf(res.Status)
	}
	db,err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	db,err = DecodeBig5(db)
	if err != nil {
		return err
	}
	return h(db)

}
func main(){
	Router.Run(":"+*port)
}
func down(){
	key := make([]byte,8)
	err := openDB(calendarDB,true,func(t *bolt.Tx)error{
		b,err := t.CreateBucketIfNotExists(calendar)
		if err != nil {
			return err
		}
		for i:=1980;i<=2100;i++{
			err =  downCalendar(i,func(db interface{})error{
				for _,d :=range rl.FindAllSubmatch(db.([]byte),-1){
					if len(d) < 3 {
						fmt.Println(string(db.([]byte)))
						continue
					}
					t,err := time.Parse(dateStamp,string(d[1]))
					if err != nil {
						//fmt.Println(err)
						t,err = time.Parse(dateStamp_,string(d[1]))
						if err != nil {
							fmt.Println(err)
							continue
						}
					}
					//v_ := rll.FindAll(d[2],-1)
					binary.BigEndian.PutUint64(key,uint64(t.Unix()))
					val := bytes.Join(rll.FindAll(d[2],-1),[]byte{','})
					b.Put(key,val)
					//fmt.Println(t,string(val))
				}
				return nil
			})
			if err != nil {
				fmt.Println(err)
				//continue
				//return err
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
