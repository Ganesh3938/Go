package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	twilio "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// User Model
type user struct {
	gorm.Model
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

//phone and otp saving in table

type Phone struct {
	gorm.Model
	PhoneNumber string `json:"phone"`
	Otp   string `json:"otp"`
}


func (user) TableName() string {
	return "user"
}

func (Phone) TableName() string {
	return "Phone"
}


var DB *gorm.DB

func ConnectDB() {
	dsn := "root:admin@tcp(localhost:3306)/webapp?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	DB = db
	fmt.Println("Database Connected Successfully")
}


// Register User
func RegisterUser(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	phone := r.FormValue("phone")
	password := r.FormValue("password")

	fmt.Println("Received Data - Name:", name, "Phone:", phone, "Password:", password)

	newUser := user{Name: name, Phone: phone, Password: password}

	result := DB.Create(&newUser)
	if result.Error != nil {
		http.Error(w, "Error registering user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func GenerateOtp() string {
	return fmt.Sprintf("%06d", rand.Intn(900000)+100000) // Ensures 6-digit OTP
}

// Send a static OTP via Twilio
func sendOTP(phoneNumber string,otp string) {
	accountSid := "AC03ece87532c421324b852da975752a24"  
	authToken := "b4876e166529c56f0f50891c4f542f16"    
	twilioPhone := "+12724221930" 

	messageBody := "Your OTP is: " + otp
    phoneNumber="+91"+phoneNumber;
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	params := &openapi.CreateMessageParams{}
	params.SetTo(phoneNumber)
	params.SetFrom(twilioPhone)
	params.SetBody(messageBody)

	resp, err := client.Api.CreateMessage(params)
	if err != nil {
		log.Println("Error sending OTP:", err)
	} else {
		log.Println("OTP sent successfully! Message SID:", *resp.Sid)
	}
}


// Login User
func LoginUser(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	phone := r.FormValue("phone")
	password := r.FormValue("password")

	var user user
	result := DB.Where("phone = ? AND password = ?", phone, password).First(&user)
	if result.Error != nil {
		// http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		http.Redirect(w, r, "/accessdenied", http.StatusSeeOther)
	}

	 otp:=GenerateOtp();

	//save Phone number and otp in the databse
	per :=Phone{
		PhoneNumber:phone,
		Otp:otp,
	}

	DB.Create(&per);
	
	//sending otp to user
	sendOTP(phone,otp);
    
	http.Redirect(w, r, "/verifyotp", http.StatusSeeOther)

	// Redirect to bookstore page after successful login
	// http.Redirect(w, r, "/bookstore", http.StatusSeeOther)

}

func VerifyOTPHandle(w http.ResponseWriter, r *http.Request) {

	r.ParseForm();
	phone := r.FormValue("phone")
	fmt.Println("Phone number=",phone);
	otp:=r.FormValue("otp");
	fmt.Println(otp);
	var phoneRecord Phone
	result := DB.Where("phone_number = ? AND otp = ?", phone, otp).First(&phoneRecord)
	fmt.Println(result);
	if result.Error != nil {
		fmt.Println("OTP Verification failed:", result.Error)
		http.Redirect(w, r, "/accessdenied", http.StatusSeeOther)
		return
	}
	fmt.Println("Deleteing the record fomr the phone table")
	deleteResult := DB.Unscoped().Where("phone_number = ?", phone).Delete(&Phone{})
	if deleteResult.Error != nil {
		fmt.Println("Failed in Deleting record");
	}else{
		fmt.Println("Record deleted");
	}
	fmt.Println("OTP Verified Successfully")

	http.Redirect(w, r, "/bookstore", http.StatusSeeOther)
	
}



// Bookstore Page
func BookStorePage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "protected/bookstore.html")
}

func AccessDeinedPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "public/accessdenied.html")
}

func VerifyOTPPage(w http.ResponseWriter , r *http.Request){
	http.ServeFile(w, r, "public/verifyotp.html")
}

// Serve Login Page
func LoginPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "public/index.html")
}

// Serve Register Page
func RegisterPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "public/userRegister.html")
}


func registerRoutes() *mux.Router {
	router := mux.NewRouter()

	// Public routes
	router.HandleFunc("/register", RegisterPage).Methods("GET")
	router.HandleFunc("/login", LoginPage).Methods("GET")
	router.HandleFunc("/accessdenied", AccessDeinedPage).Methods("GET")
	router.HandleFunc("/verifyotp",VerifyOTPPage).Methods("GET");

	// Bookstore Page
	router.HandleFunc("/bookstore", BookStorePage).Methods("GET")

	// API routes
	router.HandleFunc("/register", RegisterUser).Methods("POST")
	router.HandleFunc("/login", LoginUser).Methods("POST")
	router.HandleFunc("/verifyotp",VerifyOTPHandle).Methods("POST");

	// Set default handler for root path
	router.HandleFunc("/", LoginPage)

	// Serve files from public directory
	fileServer := http.FileServer(http.Dir("public"))
	router.PathPrefix("/").Handler(fileServer)

	return router
}




// Create Database Table
func CreateModel() {
	if !DB.Migrator().HasTable(&user{}) {
		DB.AutoMigrate(&user{})
		fmt.Println("Table created Successfully")
	} else {
		fmt.Println("User Table already exists")
	}

	if !DB.Migrator().HasTable(&Phone{}) {
		DB.AutoMigrate(&Phone{})
	} else {
		fmt.Println("Phone Table already exists")
	}

}

func main() {
	ConnectDB()
	CreateModel()

	router := registerRoutes()
	fmt.Println("Server started at http://localhost:8080")
	http.ListenAndServe(":8080", router)
}
