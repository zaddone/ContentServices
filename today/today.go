package main
import(
	"fmt"
	"net/http"
	//"io/ioutil"
	"encoding/json"
)
type Pop struct{
	Txt []string
	Img string
}

func getShanbay(h func(interface{})error)error{
	resp,err := http.Get("https://rest.shanbay.com/api/v2/quote/quotes/today/")
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf(resp.Status)
	}
	var db map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&db)
	if err != nil {
		return err
	}
	return h(&Pop{
		Txt:[]string{
			db["translation"].(string),
			db["content"].(string),
		},
		Img:db["origin_img_urls"].([]interface{})[1].(string),
	})
	//fmt.Println(string(db))
}

func getIciba(h func(interface{})error)error{
	resp,err := http.Get("http://open.iciba.com/dsapi/")
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf(resp.Status)
	}
	var db map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&db)
	if err != nil {
		return err
	}
	return h(&Pop{
		Txt:[]string{
			db["note"].(string),
			db["content"].(string),
		},
		Img:db["picture4"].(string),
	})
}

func main(){
	err := getShanbay(func(db interface{})error{
		fmt.Println(db)
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}

}
