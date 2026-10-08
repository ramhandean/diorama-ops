# Panduan Self-Hosting DioramaOps

DioramaOps dapat dijalankan mandiri di server atau VPS pribadi tanpa dependensi cloud eksternal. Seluruh data disimpan lokal dalam database SQLite WAL di dalam folder `./data`.

## Kebutuhan Sistem Minimum

- 1 vCPU
- 512 MB RAM
- 1 GB Disk Space
- Docker & Docker Compose v2

## Langkah 1: Kloning Direktori

```bash
git clone https://github.com/ramhandean/diorama-ops.git
cd diorama-ops/docker
```

## Langkah 2: Buat File Konfigurasi

Salin template konfigurasi dan atur `ADMIN_SECRET`:

```bash
cp .env.example .env
```

Edit file `.env` dan ganti nilai `ADMIN_SECRET` dengan token acak:

```bash
openssl rand -hex 24
```

## Langkah 3: Jalankan Container

```bash
docker compose up -d
```

Hub telemetri 3D Anda aktif di `http://localhost:7437`.

## Langkah 4: Hubungkan Reverse Proxy (Caddy / Nginx)

Jika menggunakan Caddy untuk HTTPS otomatis:

```caddyfile
diorama.domainanda.com {
    reverse_proxy 127.0.0.1:7437
}
```

Pastikan variabel `PUBLIC_URL` pada `.env` disesuaikan menjadi `https://diorama.domainanda.com`.

## Backup dan Pemulihan

Seluruh data tersimpan dalam direktori `./data`:
- `dioramaops.db`: Database SQLite WAL utama.
- `logos/`: File logo booth yang diunggah.
- `favicons/`: Cache favicon situs.

Untuk membuat cadangan:

```bash
tar -czvf dioramaops-backup-$(date +%F).tar.gz data/
```
