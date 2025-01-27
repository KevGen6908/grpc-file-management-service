package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"grpc-file-management-service/filemanager"
	"io/ioutil"
	"log"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := filemanager.NewFileManagerClient(conn)

	content, _ := ioutil.ReadFile("example.png")
	uploadResp, err := client.UploadFile(context.Background(), &filemanager.UploadFileRequest{
		Filename: "example.png",
		Content:  content,
	})

	if err != nil {
		log.Fatalf("could not upload file: %v", err)
	}
	fmt.Println(uploadResp.Message)

	listResp, err := client.ListFiles(context.Background(), &filemanager.ListFilesRequest{})
	if err != nil {
		log.Fatalf("could not list files: %v", err)
	}

	for _, file := range listResp.Files {
		fmt.Println("Name: %s, Create at: %s\n", file.Name, file.CreatedAt)
	}

	downloadResp, err := client.DownloadFile(context.Background(), &filemanager.DownloadFileRequest{
		Filename: "example.png",
	})
	if err != nil {
		log.Fatalf("could not download file: %v", err)
	}

	_ = ioutil.WriteFile("downloaded_example.png", downloadResp.Content, 0644)
	fmt.Printf("File downloaded successfully!\n")

}
