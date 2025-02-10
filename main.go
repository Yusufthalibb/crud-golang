package main

// Import semua package yang diperlukan
import (
	"database/sql"                       // Untuk operasi database SQL
	_ "github.com/go-sql-driver/mysql"   // Driver MySQL untuk menghubungkan ke database
	bcrypt "golang.org/x/crypto/bcrypt"  // Untuk hashing password
	"html/template"                      // Untuk memproses dan menampilkan template HTML
	"log"                                // Untuk logging (pesan error atau informasi)
	"net/http"                           // Untuk membuat server HTTP
)

// Variabel global
var db *sql.DB             // Menyimpan koneksi ke database
var err error              // Menyimpan informasi error (jika ada)
var tpl *template.Template // Menyimpan semua template HTML yang digunakan

// Fungsi init() untuk inisialisasi
func init() {
	// Membuka koneksi ke database MySQL dengan user "root" dan database "go_db"
	db, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/go_db")
	checkErr(err) // Memeriksa apakah ada error saat membuka koneksi

	// Mengecek apakah koneksi ke database aktif
	err = db.Ping()
	checkErr(err)

	// Memuat semua file HTML dari folder "html/"
	tpl = template.Must(template.ParseGlob("html/*"))
}

// Fungsi untuk memeriksa error
func checkErr(err error) {
	if err != nil {
		log.Fatalln(err) // Jika error, tampilkan pesan dan hentikan program
	}
}

// Fungsi utama untuk menjalankan server
func main() {
	// Melayani file statis seperti CSS, gambar, dll. di folder "assets"
	fs := http.FileServer(http.Dir("assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	defer db.Close() // Menutup koneksi database ketika server dihentikan

	// Rute untuk berbagai fungsi
	http.HandleFunc("/", index)               // Menampilkan daftar user
	http.HandleFunc("/userForm", userForm)    // Menampilkan form tambah user
	http.HandleFunc("/createUsers", createUsers) // Membuat user baru
	http.HandleFunc("/editUsers", editUsers)     // Menampilkan form edit user
	http.HandleFunc("/updateUsers", updateUsers) // Memperbarui data user
	http.HandleFunc("/deleteUsers", deleteUsers) // Menghapus user

	// Menjalankan server pada port 8000
	log.Println("Server is up on 8000 port")
	log.Fatalln(http.ListenAndServe(":8000", nil))
}

// Struktur data untuk tabel "users"
type user struct {
	ID        int64  // ID user
	Username  string // Username user
	FirstName string // Nama depan user
	LastName  string // Nama belakang user
	Password  []byte // Password user yang telah di-hash
}

// Menampilkan daftar user
func index(w http.ResponseWriter, req *http.Request) {
	rows, e := db.Query(
		`SELECT id,
		username,
		first_name,
		last_name,
		password
		FROM users
		ORDER BY id ASC;`) // Query untuk mengambil semua user dari database

	if e != nil {
		log.Println(e)                        // Log error jika query gagal
		http.Error(w, e.Error(), http.StatusInternalServerError)
		return
	}

	users := make([]user, 0) // Array untuk menyimpan data user
	for rows.Next() {
		usr := user{} // Variabel sementara untuk setiap baris data
		err := rows.Scan(&usr.ID, &usr.Username, &usr.FirstName, &usr.LastName, &usr.Password)
		if err != nil {
			log.Println(err) // Log error jika scan data gagal
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, usr) // Menambahkan data user ke array
	}

	log.Println(users) // Log data user
	tpl.ExecuteTemplate(w, "index.html", users) // Menampilkan template dengan data user
}

// Menampilkan form untuk menambah user baru
func userForm(w http.ResponseWriter, req *http.Request) {
	err = tpl.ExecuteTemplate(w, "userForm.html", nil) // Menampilkan template form
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // Tampilkan error jika gagal
		return
	}
}

// Menambahkan user baru ke database
func createUsers(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost { // Memastikan hanya menerima metode POST
		usr := user{}
		usr.Username = req.FormValue("username") // Mengambil input dari form
		usr.FirstName = req.FormValue("firstName")
		usr.LastName = req.FormValue("lastName")
		bPass, e := bcrypt.GenerateFromPassword([]byte(req.FormValue("password")), bcrypt.MinCost) // Hash password

		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError) // Tampilkan error jika hashing gagal
			return
		}
		usr.Password = bPass

		// Menyimpan data user ke database
		_, e = db.Exec("INSERT INTO users (username,first_name, last_name, password) VALUES (?,?,?,?)", usr.Username,
			usr.FirstName,
			usr.LastName,
			usr.Password,
		)

		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}

		// Redirect ke halaman utama setelah berhasil
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not supported", http.StatusMethodNotAllowed) // Jika bukan POST, tampilkan error
}

// Menampilkan form edit user berdasarkan ID
func editUsers(w http.ResponseWriter, req *http.Request) {
	id := req.FormValue("id") // Mengambil ID dari parameter
	rows, err := db.Query(
		`SELECT id,
	 	username,
		first_name,
		last_name
		FROM users
		WHERE id = ` + id + `;`) // Query untuk mendapatkan data user berdasarkan ID

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	usr := user{}
	for rows.Next() {
		rows.Scan(&usr.ID, &usr.Username, &usr.FirstName, &usr.LastName) // Scan data ke dalam variabel user
	}
	tpl.ExecuteTemplate(w, "editUser.html", usr) // Menampilkan template form edit
}

// Memperbarui data user di database
func updateUsers(w http.ResponseWriter, req *http.Request) {
	_, er := db.Exec("UPDATE users set username = ?, first_name = ?, last_name = ? WHERE id = ?",
		req.FormValue("username"),   // Username baru
		req.FormValue("firstName"),  // Nama depan baru
		req.FormValue("lastName"),   // Nama belakang baru
		req.FormValue("id"),         // ID user
	)

	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError) // Tampilkan error jika update gagal
		return
	}

	http.Redirect(w, req, "/", http.StatusSeeOther) // Redirect ke halaman utama
}

// Menghapus user berdasarkan ID
func deleteUsers(w http.ResponseWriter, req *http.Request) {
	id := req.FormValue("id") // Mengambil ID dari parameter

	if id == "" {
		http.Error(w, "Please Send ID", http.StatusBadRequest) // Error jika ID tidak ada
		return
	}

	// mengapus data user dari database
	_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // Tampilkan error jika gagal
		return
	}

	// mereset autoinkrement jika tabel kosong
	_, err = db.Exec("ALTER TABLE users AUTO_INCREMENT = 1")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, "/", http.StatusSeeOther) // Redirect ke halaman utama
}
