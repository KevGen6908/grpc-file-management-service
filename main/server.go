package main

import (
	"context"
	"fmt"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"grpc-file-management-service/filemanager"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	storageDir           = "uploads"
	maxUploadConnections = 10
	maxListConnections   = 100
)

type fileManagerServer struct {
	filemanager.UnimplementedFileManagerServer
	uploadLimiter *semaphore.Weighted
	listLimiter   *semaphore.Weighted
	mu            sync.Mutex
}

func NewFileManagerServer() *fileManagerServer {
	return &fileManagerServer{
		uploadLimiter: semaphore.NewWeighted(maxUploadConnections),
		listLimiter:   semaphore.NewWeighted(maxListConnections),
	}
}

func (s *fileManagerServer) UploadFile(ctx context.Context, request *filemanager.UploadFileRequest) (*filemanager.UploadFileResponse, error) {
	if err := s.uploadLimiter.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("too many concurrent uploads: %w", err)
	}
	defer s.uploadLimiter.Release(1)

	filePath := filepath.Join(storageDir, request.Filename)
	err := ioutil.WriteFile(filePath, request.Content, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	log.Printf("uploaded file: %s", request.Filename)
	return &filemanager.UploadFileResponse{Message: "File uploaded successfully"}, nil
}

func (s *fileManagerServer) ListFiles(ctx context.Context, request *filemanager.ListFilesRequest) (*filemanager.ListFilesResponse, error) {
	if err := s.listLimiter.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("too many concurrent uploads: %w", err)
	}
	defer s.listLimiter.Release(1)

	files, err := ioutil.ReadDir(storageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var metadata []*filemanager.FileMetadata
	for _, file := range files {
		if !file.IsDir() {
			metadata = append(metadata, &filemanager.FileMetadata{
				Name:      file.Name(),
				CreatedAt: file.ModTime().Format(time.RFC3339),
				UpdatedAt: file.ModTime().Format(time.RFC3339),
			})
		}
	}

	return &filemanager.ListFilesResponse{Files: metadata}, nil
}

func (s *fileManagerServer) DownloadFile(ctx context.Context, request *filemanager.DownloadFileRequest) (*filemanager.DownloadFileResponse, error) {
	if err := s.uploadLimiter.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("too many concurrent downloads: %w", err)
	}
	defer s.uploadLimiter.Release(1)

	filePath := filepath.Join(storageDir, request.Filename)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(`file "%s" not found`, request.Filename)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	log.Printf("downloaded file: %s", request.Filename)
	return &filemanager.DownloadFileResponse{Content: content}, nil
}

func main() {
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		log.Fatalf("failed to create storage directory: %v", err)
	}

	server := grpc.NewServer()
	filemanager.RegisterFileManagerServer(server, NewFileManagerServer())

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("Server is running on port 50051")
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
