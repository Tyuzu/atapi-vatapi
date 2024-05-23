package main

import (
    "fmt"
    "net/http"
    "log"
	"io"
	"os"
	"time"
	"os/exec"
	"crypto/rand"
	rndm "math/rand"
	"crypto/md5"
	"path/filepath"

    "github.com/julienschmidt/httprouter"
)

// Gif - We will be using this Gif type to perform crud operations
type GIF struct {
	Title  string
	Author string
	Tags   []string
	Date   string
	URL    string
	Views  int
	Likes  int
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}


func UploadVideoFileHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	enableCors(&w)
	if r.Method == "POST" {           
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			fmt.Printf("Could not parse multipart form: %v\n", err)
			renderError(w, "CANT_PARSE_FORM", http.StatusInternalServerError)
				return
		}
		fmt.Println("fbdh g: ",r.FormValue("csrftoken"))
		var count int = 0
		var fileEndings string
		var folderpath string
		var fileName string
		var postType string
		var postName string = GenerateName(12)

			fmt.Println("hao")
			file, fileHeader, err := r.FormFile("myfile")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Println("hao")
			file, err = fileHeader.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			defer file.Close()
			fmt.Println("file OK")
			count++
			//~ fmt.Println(count)
			fileSize := fileHeader.Size
			fmt.Printf("File size (bytes): %v\n", fileSize)
			// validate file size
			if fileSize > maxUploadSize {
				renderError(w, "FILE_TOO_BIG", http.StatusBadRequest)
				return
			}
			fileBytes, err := io.ReadAll(file)
			if err != nil {
				renderError(w, "INVALID_FILE"+http.DetectContentType(fileBytes), http.StatusBadRequest)
				return
			}
			//~ // check file type, detectcontenttype only needs the first 512 bytes
			detectedFileType := http.DetectContentType(fileBytes)
			switch detectedFileType {
			case "video/mp4":
				fileEndings = ".mp4"
				folderpath = "./videos"
				postType = "v"
				fileName = fmt.Sprintf("v%v_%d",postName,count)
				break
			case "video/webm":
				fileEndings = ".webm"
				folderpath = "./videos"
				postType = "v"
				fileName = fmt.Sprintf("v%v_%d",postName,count)
				break
			case "image/jpg", "image/jpeg", "image/png", "image/webp":
				fileEndings = ".png"
				folderpath = "./images"
				postType = "i"
				fileName = fmt.Sprintf("i%v_%d",postName,count)
				break
			default:
				renderError(w, "INVALID_FILE_TYPE", http.StatusBadRequest)
				return
			}
			// if fileName exists in Redis, again GenerateName(rndmToken(12))
			//		fileEndings, err := mime.ExtensionsByType(detectedFileType)
			//~ fileName = GenerateName(16)
			if err != nil {
				renderError(w, "CANT_READ_FILE_TYPE", http.StatusInternalServerError)
				return
			}
			newFileName := fileName + fileEndings
			fmt.Println("fdeshyfu regfu egyure gyre u", newFileName)
			newPath := filepath.Join(folderpath, newFileName)
			fmt.Printf("FileType: %s, File: %s\n", detectedFileType, newPath)

			// write file
			newFile, err := os.Create(newPath)
			if err != nil {
				renderError(w, "CANT_WRITE_FILE", http.StatusInternalServerError)
				return
			}
			defer newFile.Close() // idempotent, okay to call twice
			if _, err := newFile.Write(fileBytes); err != nil || newFile.Close() != nil {
				renderError(w, "CANT_WRITE_FILE", http.StatusInternalServerError)
				return
			}
			if postType == "v" {
				FFConvert(fileName , fileEndings )
			}
			postName = fmt.Sprintf("%v%v", postType,postName)
			//~ fmt.Fprintf(w, "{\"postname\": \""+postName+"\", \"postcount\": "+fmt.Sprintf("%d",count)+"}")
			//~ fmt.Fprintf(w, "{\"postname\":\""+postName+"\", \"postcount\":"+fmt.Sprintf("%d",count)+", \"posttype\":\""+ fmt.Sprintf("%s",postType) +"\"}")
			//~ var res = fmt.Sprintf("{\"postname\":\""+postName+"\", \"postcount\":"+fmt.Sprintf("%v",count)+", \"posttype\":\""+postType+"\"}")
			
			var res = fmt.Sprintf("{\"postname\": \""+postName+"\", \"postcount\": "+fmt.Sprintf("%v",count)+", \"posttype\": \""+postType+"\"}")
			fmt.Fprintf(w,res)
			rdxHset("posts", postName, res)
	}
}

func FFConvert(fileName string, fileEndings string) {
	getFrom := "./videos" + "/" + fileName + fileEndings
	saveAs := "./streams" + "/" + fileName + ".mp4"
	cmd := exec.Command("ffmpeg", "-i", getFrom, "-filter:v", "scale=-2:480:flags=lanczos", "-c:a", "copy", "-pix_fmt", "yuv420p", saveAs)
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Waiting for command to finish...")
	err = cmd.Wait()
	log.Printf("Command finished with error: %v", err)
	FFPoster(fileName)
}


func FFPoster(fileName string) {
	getFrom := "./streams" + "/" + fileName + ".mp4"
	log.Printf(getFrom)
	saveAs := "./images/" + fileName + ".jpg"
	log.Printf(saveAs)
	cmd := exec.Command("ffmpeg", "-i", getFrom, "-vf", "scale=-2:480:flags=lanczos", saveAs)
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Waiting for command to finish...")
	err = cmd.Wait()
	log.Printf("Command finished with error: %v", err)
}


func sendImageAsBytes(w http.ResponseWriter, r *http.Request, a httprouter.Params) {
	buf, err := os.ReadFile("./images/"+a.ByName("imageName"))
	if err != nil {
		log.Print(err)
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(buf)
}


func CSRF(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	enableCors(&w)
	fmt.Fprintf(w,GenerateName(8))
}


func EventNew(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	enableCors(&w)
	var wht = r.FormValue("what")
	var whr = r.FormValue("where")
	var whns = r.FormValue("whens")
	var whne = r.FormValue("whene")
	var whnst = r.FormValue("whenst")
	var whnet = r.FormValue("whenet")
	var typ = r.FormValue("typ")
	var cat = r.FormValue("cat")
	var desc = r.FormValue("desc")
	var eventid = GenerateName(18)
	var res = fmt.Sprintf("{\"what\":\""+wht+"\", \"whens\":\""+whns+"\", \"whene\":\""+whne+"\", \"whenst\":\""+whnst+"\", \"whenet\":\""+whnet+"\", \"typ\":\""+typ+"\", \"cat\":\""+cat+"\", \"desc\":\""+desc+"\", \"where\":\""+whr+"\", \"eventid\":\""+eventid+"\"}")
	rdxHset("events", eventid, res)
	fmt.Fprintf(w,res)
}

func EventView(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	enableCors(&w)
	var eventid = r.URL.Query().Get("eventid")
	res, _ := rdxHget("events", eventid)
	fmt.Fprintf(w,res)
}


func PlaceNew(w http.ResponseWriter, r *http.Request, k httprouter.Params) {
	enableCors(&w)
	var nameofplace = r.FormValue("nameofplace")
	var address = r.FormValue("address")
	var category = r.FormValue("category")
	var closingtime = r.FormValue("closingtime")
	var openingtime = r.FormValue("openingtime")
	var phonenumber = r.FormValue("phonenumber")
	var instagaram = r.FormValue("instagaram")
	var website = r.FormValue("website")
	var facilities = r.FormValue("facilities")
	var about = r.FormValue("about")
	var paymentmethod = r.FormValue("paymentmethod")
	var keyword = r.FormValue("keyword")
	var placeid = GenerateName(18)
	var res = fmt.Sprintf("{\"nameofplace\":\""+nameofplace+"\", \"address\":\""+address+"\", \"category\":\""+category+"\", \"closingtime\":\""+closingtime+"\", \"openingtime\":\""+openingtime+"\", \"phonenumber\":\""+phonenumber+"\", \"instagaram\":\""+instagaram+"\", \"facilities\":\""+facilities+"\", \"about\":\""+about+"\", \"paymentmethod\":\""+paymentmethod+"\", \"keyword\":\""+keyword+"\", \"website\":\""+website+"\", \"placeid\":\""+placeid+"\"}")
	rdxHset("places", placeid, res)
	fmt.Fprintf(w,res)
}

func PlaceView(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	enableCors(&w)
	var placeid = r.URL.Query().Get("placeid")
	res, _ := rdxHget("places", placeid)
	fmt.Fprintf(w,res)
}

func PostView(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	enableCors(&w)
	var postid = r.URL.Query().Get("postid")
	res, _ := rdxHget("posts", postid)
	fmt.Fprintf(w,res)
}

func Res(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	enableCors(&w)
	var translated = "trns"
	var lang = "EN"
	fmt.Fprintf(w,"{\"translated\":\""+translated+"\", \"lang\":\""+lang+"\"}")
}

func Translate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	enableCors(&w)
	var translated = r.FormValue("trns")
	var lang = "EN"
	fmt.Fprintf(w, "{\"translated\":\""+translated+"\", \"lang\":\""+lang+"\"}")
}

/*
func UploadVideoFileHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	enableCors(&w)
	if r.Method == "POST" {           
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			fmt.Printf("Could not parse multipart form: %v\n", err)
			renderError(w, "CANT_PARSE_FORM", http.StatusInternalServerError)
		}
		var postname string = GenerateName(12)
		var fileEndings string
		var fileName string
		var folderpath string

		files := r.MultipartForm.File["file"]
		for filecount, fileHeader := range files {
			fmt.Println(filecount)
			log.Println("hao")
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			defer file.Close()
			log.Println("file OK")
			title := r.FormValue("title")
			tags := strings.ToLower(r.FormValue("tags"))
			fmt.Println(tags)
			// Get and print outfile size
			
			f := func(c rune) bool {
			return !unicode.IsLetter(c) && !unicode.IsNumber(c)
			}
			titleArr := strings.FieldsFunc(strings.ToLower(title), f)
			fmt.Printf("Fields are: %q", titleArr)
			fileSize := fileHeader.Size
			fmt.Printf("File size (bytes): %v\n", fileSize)
			// validate file size
			if fileSize > maxUploadSize {
				renderError(w, "FILE_TOO_BIG", http.StatusBadRequest)
			}
			fileBytes, err := io.ReadAll(file)
			if err != nil {
				renderError(w, "INVALID_FILE"+http.DetectContentType(fileBytes), http.StatusBadRequest)
			}
			
			//~ // check file type, detectcontenttype only needs the first 512 bytes
			detectedFileType := http.DetectContentType(fileBytes)
			switch detectedFileType {
			case "video/mp4":
				fileEndings = ".mp4"
				folderpath = "./videos"
				break
			case "video/webm":
				fileEndings = ".webm"
				folderpath = "./videos"
				break
			case "image/gif":
				fileEndings = ".gif"
				folderpath = "./images"
				break
			case "image/png":
				fileEndings = ".png"
				folderpath = "./images"
				break
			case "image/webp":
				fileEndings = ".webp"
				folderpath = "./images"
				break
			case "image/jpg":
				fileEndings = ".jpg"
				folderpath = "./images"
				break
			case "image/jpeg":
				fileEndings = ".jpeg"
				folderpath = "./images"
				break
			default:
				renderError(w, "INVALID_FILE_TYPE", http.StatusBadRequest)
			}
			fileName = r.FormValue("key")
			fmt.Println("fileName : ", fileName)
			if err != nil {
				renderError(w, "CANT_READ_FILE_TYPE", http.StatusInternalServerError)
			}
			//~ newFileName := fileName + fileEndings
			//~ newPath := filepath.Join(uploadPath, newFileName)
			//~ newFileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), fileEndings)
			//~ newFileName := GenerateName(16)+fmt.Sprintf("%d",filecount)+fileEndings
			newFileName := postname+fmt.Sprintf("%d",filecount)+fileEndings
			fmt.Println(newFileName)
			newPath := filepath.Join(folderpath, newFileName)
			//~ newPath := fmt.Sprintf("./images/%d%s", time.Now().UnixNano(), filepath.Ext(fileHeader.Filename))
			//~ fmt.Printf("FileType: %s, File: %s\n", detectedFileType, newPath)

			fmt.Printf("Uploaded File: %+v\n", fileHeader.Filename)
			fmt.Printf("File Size: %+v\n", fileHeader.Size)
			fmt.Printf("MIME Header: %+v\n", fileHeader.Header)
			// write file
			newFile, err := os.Create(newPath)
			if err != nil {
				renderError(w, "CANT_WRITE_FILE", http.StatusInternalServerError)
			}
			defer newFile.Close() // idempotent, okay to call twice
			if _, err := newFile.Write(fileBytes); err != nil || newFile.Close() != nil {
				renderError(w, "CANT_WRITE_FILE", http.StatusInternalServerError)
			}
		}
			fmt.Fprintf(w, postname)
//				tmpl.ExecuteTemplate(w, "show.html", fileName)
	}
}*/

func GenerateName(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789_ABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rndm.Intn(len(letters))]
	}
	return string(b)
}

func init() {
	rndm.Seed(time.Now().UnixNano())
}


func renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}

func rndmToken(len int) int64 {
	b := make([]byte, len)
	n, _ := rand.Read(b)
	return int64(n)
}

func EncrypIt(strToHash string) string {
	data := []byte(strToHash)
	return fmt.Sprintf("%x", md5.Sum(data))
}

func SessionVerify(sessionKey string) string {
	return fmt.Sprintf(sessionKey)
}

func Ignore(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	http.ServeFile(w, r, "favicon.png")
}

