package main

import (
	"bytes"
	"context"
	"google.golang.org/grpc"
	"grpc-file-management-service/filemanager"
	"html/template"
	"io"
	"log"
	"net/http"
)

var grpcClient filemanager.FileManagerClient
var serverPort = "8081"

const tpl = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>File Management</title>
</head>
<body>
    <h1>File Management</h1>
    <h2>Upload File</h2>
    <form action="/upload" method="POST" enctype="multipart/form-data">
        <input type="file" name="file">
        <button type="submit">Upload</button>
    </form>
    <h2>Files</h2>
    <ul>
        {{range .Files}}
            <li>
                <a href="/download?filename={{.Name}}">{{.Name}}</a>
                (Created: {{.CreatedAt}})
            </li>
        {{else}}
            <p>No files found</p>
        {{end}}
    </ul>
</body>
</html>
`

func listFiles(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	resp, err := grpcClient.ListFiles(ctx, &filemanager.ListFilesRequest{})
	if err != nil {
		http.Error(w, "Failed to list files", http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.New("webpage").Parse(tpl))
	tmpl.Execute(w, resp)
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	_, err = grpcClient.UploadFile(context.Background(), &filemanager.UploadFileRequest{
		Filename: header.Filename,
		Content:  buf.Bytes(),
	})
	if err != nil {
		http.Error(w, "Failed to upload file", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, "Missing filename", http.StatusBadRequest)
		return
	}

	resp, err := grpcClient.DownloadFile(context.Background(), &filemanager.DownloadFileRequest{
		Filename: filename,
	})

	if err != nil {
		http.Error(w, "Failed to download file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write(resp.Content)
}

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatal("Failed to connect to gRPC server")
	}
	defer conn.Close()

	grpcClient = filemanager.NewFileManagerClient(conn)

	http.HandleFunc("/", listFiles)
	http.HandleFunc("/upload", uploadFile)
	http.HandleFunc("/download", downloadFile)

	port := serverPort // Используем заранее заданный порт
	log.Printf("Starting web server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}

}
