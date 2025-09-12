package main

import (
    "context"
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    "strings"

    _ "github.com/jackc/pgx/v5/stdlib"
)

func hashBytes(b []byte) string {
    h := sha256.Sum256(b)
    return hex.EncodeToString(h[:])
}

func main() {
    dsn := os.Getenv("PS_PG_DSN")
    if dsn == "" {
        log.Fatal("PS_PG_DSN is required")
    }
    outDir := flag.String("out", "blobs/rulepacks", "output directory for YAML/DSL blobs")
    backend := flag.String("backend", "db", "backend tag to set (db|s3|gcs)")
    s3Bucket := flag.String("s3-bucket", "", "optional S3 bucket name (for URL fill only)")
    s3Prefix := flag.String("s3-prefix", "", "optional S3 key prefix (for URL fill only)")
    flag.Parse()

    if err := os.MkdirAll(*outDir, 0o755); err != nil { log.Fatal(err) }

    db, err := sql.Open("pgx", dsn)
    if err != nil { log.Fatal(err) }
    defer db.Close()

    ctx := context.Background()
    rows, err := db.QueryContext(ctx, `SELECT id, yaml_content FROM rulepack_versions WHERE yaml_content IS NOT NULL`)
    if err != nil { log.Fatal(err) }
    defer rows.Close()

    count := 0
    for rows.Next() {
        var id string
        var yaml string
        if err := rows.Scan(&id, &yaml); err != nil { log.Fatal(err) }
        b := []byte(yaml)
        sum := hashBytes(b)
        rel := filepath.Join(*outDir, sum+".yaml")
        if err := ioutil.WriteFile(rel, b, 0o644); err != nil { log.Fatal(err) }

        // Compute URL hint
        var url string
        switch *backend {
        case "s3":
            if *s3Bucket == "" {
                log.Printf("warning: s3 backend selected but no bucket set; setting file:// URL")
                url = "file://" + filepath.ToSlash(rel)
            } else {
                key := strings.TrimPrefix(filepath.ToSlash(*s3Prefix+"/"+sum+".yaml"), "/")
                url = fmt.Sprintf("s3://%s/%s", *s3Bucket, key)
            }
        case "gcs":
            // Just a URL hint; upload handled externally
            url = fmt.Sprintf("gs://%s/%s/%s.yaml", *s3Bucket, strings.TrimPrefix(*s3Prefix, "/"), sum)
        default:
            url = "file://" + filepath.ToSlash(rel)
        }

        // Update DB metadata (hash, url, backend)
        if _, err := db.ExecContext(ctx,
            `UPDATE rulepack_versions SET yaml_hash=$1, yaml_blob_url=$2, storage_backend=$3 WHERE id=$4`,
            sum, url, *backend, id,
        ); err != nil { log.Fatal(err) }
        count++
    }
    if err := rows.Err(); err != nil { log.Fatal(err) }
    log.Printf("processed %d rulepack version blobs", count)
}

