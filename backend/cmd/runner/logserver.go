package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	logkitv1 "ila/backend/gen/logkit/v1"
	"ila/backend/internal/logingest"
)

// jalankanPenerimaLog membuka server gRPC penerima log Engine.
//
// Alamat kosong berarti penerimaan dimatikan: Engine tetap menulis log lokal
// sendiri, jadi tidak ada yang hilang secara permanen.
func jalankanPenerimaLog(addr, dir, certFile, keyFile, caFile string, log *slog.Logger) (*grpc.Server, error) {
	if addr == "" {
		log.Info("penerima log engine dimatikan", "evt", "log.server.mati")
		return nil, nil
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("mendengar %s: %w", addr, err)
	}

	opsi := []grpc.ServerOption{}
	if certFile != "" && keyFile != "" {
		kredensial, err := kredensialServer(certFile, keyFile, caFile)
		if err != nil {
			_ = lis.Close()
			return nil, err
		}
		opsi = append(opsi, grpc.Creds(kredensial))
	} else {
		// Tanpa sertifikat hanya boleh untuk pengembangan lokal.
		log.Warn("penerima log tanpa mTLS", "evt", "log.server.tanpa-mtls", "addr", addr)
	}

	srv := grpc.NewServer(append(opsi,
		// Engine mengirim ping keepalive tiap 30 detik. Tanpa kebijakan ini
		// gRPC menolak ping lebih cepat dari 5 menit (ENHANCE_YOUR_CALM /
		// too_many_pings) sehingga stream log terputus lalu tersambung lagi
		// setiap ~2 menit.
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)...)
	penyimpan := logingest.Baru(dir, log)
	logkitv1.RegisterLogServiceServer(srv, penyimpan)

	go func() {
		log.Info("penerima log engine berjalan", "evt", "log.server.aktif", "addr", addr, "dir", dir)
		if err := srv.Serve(lis); err != nil {
			log.Error("penerima log berhenti tak terduga", "evt", "log.server.berhenti", "error", err.Error())
		}
		_ = penyimpan.Tutup()
	}()

	return srv, nil
}

// kredensialServer menyiapkan mTLS dua arah: klien wajib menyerahkan sertifikat
// yang ditandatangani CA kita, sehingga pihak lain tidak bisa menyuntik log.
func kredensialServer(certFile, keyFile, caFile string) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("memuat sertifikat server: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	if caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("membaca CA klien: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("CA klien tidak bisa dibaca")
		}
		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return credentials.NewTLS(cfg), nil
}
