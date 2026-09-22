package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/forg3/esocial-emissor-livre/internal/storage"
	"github.com/forg3/esocial-emissor-livre/internal/web"
)

func abrirNavegador(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Aviso: abertura automática do navegador não suportada neste ambiente: %v", err)
	}
}

func main() {
	porta := flag.Int("porta", 8000, "Porta HTTP do servidor web")
	dadosDir := flag.String("dados", "./dados", "Diretório de dados e banco SQLite local")
	semNavegador := flag.Bool("sem-navegador", false, "Não abrir o navegador automaticamente")
	flag.Parse()

	dbPath := filepath.Join(*dadosDir, "esocial.db")
	db, err := storage.Abrir(dbPath)
	if err != nil {
		log.Fatalf("Erro ao inicializar banco local SQLite em %s: %v", dbPath, err)
	}
	defer db.Fechar()

	srvWeb, err := web.NovoServidor(db)
	if err != nil {
		log.Fatalf("Erro ao instanciar servidor web: %v", err)
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", *porta),
		Handler:      srvWeb.Rotas(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	urlAcesso := fmt.Sprintf("http://localhost:%d", *porta)

	go func() {
		log.Printf("=======================================================")
		log.Printf("  Validador eSocial - Servidor Web Ativo")
		log.Printf("  Interface: %s", urlAcesso)
		log.Printf("  Banco de dados: %s", dbPath)
		log.Printf("  Pressione Ctrl+C para encerrar com segurança.")
		log.Printf("=======================================================")

		if !*semNavegador {
			go func() {
				time.Sleep(350 * time.Millisecond)
				abrirNavegador(urlAcesso)
			}()
		}

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Falha crítica no servidor HTTP: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Encerrando servidor com segurança...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Erro durante o desligamento: %v", err)
	}
	log.Println("Servidor finalizado com sucesso.")
}
