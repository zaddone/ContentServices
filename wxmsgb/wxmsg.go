package wxmsgb

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	//"os"
	"strings"
	//"io/ioutil"
	"github.com/zaddone/studySystem/request"
	//"github.com/zaddone/studySystem/conf"
	"encoding/json"
	//"path/filepath"
	"sync"
	"time"
)

var (
	AppId        = ""
	Sec          = ""
	env   string = ""

	wxToKenUrl = "https://api.weixin.qq.com/cgi-bin/token"
	//wxToKenUrl= "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=wx92ebd09c7b0d944f&secret=b3005d3c298e27b60ee1f90d188a9d86"
	toKen   string
	TimeOut int64
	//env string = "guomi-2i7wu"
	//MaxCount float64 = 10000
	//ExpiresIn int64
	R sync.Mutex
)

func setToken() int64 {
	R.Lock()
	db := map[string]interface{}{}
	err := request.ClientHttp(wxToKenUrl, "GET", []int{200}, nil, func(body io.Reader) error {
		return json.NewDecoder(body).Decode(&db)
	})
	R.Unlock()
	if err != nil {
		return setToken()
	}
	if db["access_token"] == nil {
		fmt.Println(db)
		//time.Sleep(1 * time.Second)
		return setToken()
	}
	toKen = db["access_token"].(string)
	return int64(db["expires_in"].(float64)) - 100

}

func Reload(appid,sec,e string) {
	AppId = appid
	Sec = sec
	env = e
	wxToKenUrl = fmt.Sprintf("%s?%s", wxToKenUrl,
		(&url.Values{
			"grant_type": []string{"client_credential"},
			"appid":      []string{AppId},
			"secret":     []string{Sec},
		}).Encode())
}

func GetToken() string {
	if TimeOut > time.Now().Unix() {
		return toKen
	}
	TimeOut = setToken() + time.Now().Unix()
	return toKen
}

func PostRequest(url string, PostBody map[string]interface{}, h func(io.Reader) error) error {

	url = fmt.Sprintf("%s?access_token=%s", url, GetToken())
	PostBody["env"] = env
	db, err := json.Marshal(PostBody)
	if err != nil {
		panic(err)
	}
	res,err := http.Post(url,"application/json",bytes.NewReader(db))
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf(res.Status)
	}
	defer res.Body.Close()
	return h(res.Body)

	//fmt.Println(string(db))
	//return request.ClientHttp_(url, "POST", bytes.NewReader(db), http.Header{"Content-Type": []string{"application/x-www-form-urlencoded", "multipart/form-data"}}, func(body io.Reader, st int) error {
	//	if st == 200 {
	//		return h(body)
	//	}
	//	var da [8192]byte
	//	n, err := body.Read(da[:])
	//	return fmt.Errorf("status code %d %s %s", st, url, string(da[:n]), err)
	//})

}

func DeleteColl(c_name string) error {

	return PostRequest("https://api.weixin.qq.com/tcb/databasecollectiondelete", map[string]interface{}{"collection_name": c_name}, func(body io.Reader) error {
		var res map[string]interface{}
		json.NewDecoder(body).Decode(&res)
		if res["errcode"].(float64) == 0 {
			return nil
		}
		//fmt.Println(res,res["errcode"].(float64),res["errmsg"].(string))
		return fmt.Errorf(res["errmsg"].(string))
	})

}
func CreateColl(c_name string) error {

	return PostRequest("https://api.weixin.qq.com/tcb/databasecollectionadd", map[string]interface{}{"collection_name": c_name}, func(body io.Reader) error {
		var res map[string]interface{}
		json.NewDecoder(body).Decode(&res)
		if res["errcode"].(float64) == 0 {
			return nil
		}
		//fmt.Println(res,res["errcode"].(float64),res["errmsg"].(string))
		return fmt.Errorf(res["errmsg"].(string))
	})

}

func DBDelete(coll string, ids []string) error {
	fmt.Println(ids)
	return PostRequest(
		"https://api.weixin.qq.com/tcb/databasedelete",
		map[string]interface{}{
			"query": fmt.Sprintf(
				"db.collection(\"%s\").where({_id:db.command.in([%s])}).remove()",
				coll,
				//config.Conf.CollPageName,
				strings.Join(ids, ","))},
		func(body io.Reader) error {

			var res map[string]interface{}
			json.NewDecoder(body).Decode(&res)
			errcode := res["errcode"].(float64)
			if errcode == 0 {
				return nil
			}
			return fmt.Errorf("%.0f %s", errcode, res["errmsg"].(string))
		})
}
func UploadWX(coll string,data io.Reader)error{
	fileName:= fmt.Sprintf("tmp/%s/%d",coll,time.Now().Unix())
	//fmt.Println(fileName)
	var res map[string]interface{}
	err := PostRequest(
		"https://api.weixin.qq.com/tcb/uploadfile",
		map[string]interface{}{
			"path": fileName,
		},
		func(body io.Reader) error {
			return json.NewDecoder(body).Decode(&res)
		},
	)
	if err != nil {
		//panic(err)
		return err
	}
	fmt.Println(res)
	//var b bytes.Buffer
	b := &bytes.Buffer{}
	w := multipart.NewWriter(b)
	w.WriteField("key",fileName)
	w.WriteField("Signature",res["authorization"].(string))
	w.WriteField("x-cos-security-token",res["token"].(string))
	w.WriteField("x-cos-meta-fileid",res["cos_file_id"].(string))

	fw,err := w.CreateFormFile("file",fileName)
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, data); err != nil {
		return err
	}
	w.Close()
	err = Upload_(
		res["url"].(string),
		b,
		w.FormDataContentType(),
	)
	if err != nil {
		//panic(err)
		return err
	}
	return UpDBToWX_(coll,fileName,res["file_id"].(string))

}

func UpDBToWX_(coll,fp, pid string) error {
	var res map[string]interface{}
	err := PostRequest(
		"https://api.weixin.qq.com/tcb/databasemigrateimport",
		map[string]interface{}{
			"collection_name": coll,
			"file_path":       fp,
			"file_type":       1,
			"stop_on_error":   false,
			"conflict_mode":   2,
		},
		func(body io.Reader) error {
			return json.NewDecoder(body).Decode(&res)
		})
	if err != nil {
		panic(err)
		return err
	}
	fmt.Println("databasemigrateimport", res)
	if res["errcode"].(float64) != 0 {
		return fmt.Errorf(res["errmsg"].(string))
	}
	job_id := res["job_id"]
	for {
		<-time.After(5 * time.Second)
		err = PostRequest(
			"https://api.weixin.qq.com/tcb/databasemigratequeryinfo",
			map[string]interface{}{
				"job_id": job_id,
			},
			func(body io.Reader) error {
				return json.NewDecoder(body).Decode(&res)
			})
		if err != nil {
			return err
			//log.Println(err)
		}
		fmt.Println("info", res)
		if res["errcode"].(float64) != 0 {
			return fmt.Errorf(res["errmsg"].(string))
		}

		if strings.EqualFold(res["status"].(string), "fail") {
			continue
			panic(res)
		}
		if strings.EqualFold(res["status"].(string), "success") {
			fmt.Println(res)
			break
		}

	}
	err = PostRequest(
		"https://api.weixin.qq.com/tcb/batchdeletefile",
		map[string]interface{}{
			"fileid_list": []string{pid},
		},
		func(body io.Reader) error {
			return json.NewDecoder(body).Decode(&res)
		})

	fmt.Println("del", res)
	if res["errcode"].(float64) != 0 {
		fmt.Println(res)
		return fmt.Errorf(res["errmsg"].(string))
	}
	return nil
	//return os.Remove(uri)

}

func FuncWXDB(name string, body io.Reader, f func(interface{}) error) error {

	u := url.Values{}
	u.Set("access_token", GetToken())
	u.Set("env", env)
	u.Set("name", name)

	res, err := http.Post(fmt.Sprintf("https://api.weixin.qq.com/tcb/invokecloudfunction?%s", u.Encode()), "application/x-www-form-urlencoded", body)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("%d:%s", res.StatusCode, res.Status)
	}
	var db interface{}
	err = json.NewDecoder(res.Body).Decode(&db)
	res.Body.Close()
	if err != nil {
		return err
	}
	return f(db)

}
func UpdateWXDB(coll string, _id string, body string) error {
	//fmt.Println(body)
	var res map[string]interface{}
	err := PostRequest(
		"https://api.weixin.qq.com/tcb/databaseupdate",
		map[string]interface{}{
			"query": fmt.Sprintf("db.collection(\"%s\").doc(\"%s\").set({data:%s})", coll, _id, body)},
		func(body io.Reader) error {
			return json.NewDecoder(body).Decode(&res)
		})
	if err != nil {
		return err
	}
	if res["errcode"].(float64) != 0 {
		return fmt.Errorf("%.0f %s", res["errcode"].(float64), res["errmsg"].(string))
	}
	return nil
}



func Upload_(url string, body io.Reader,ContentType string) (err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", ContentType)
	//fmt.Println()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	fmt.Println(res.Request)
	if res.StatusCode != 204 {
		//fmt.Println(res.StatusCode,res.Status)
		db, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return err
		}
		fmt.Println(string(db))
		return fmt.Errorf(res.Status)
	}
	return
}

