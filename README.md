# InstantGate API

**InstantGate** herhangi bir ilişkisel veritabanını (MySQL/PostgreSQL) saniyeler içinde tam fonksiyonlu bir REST API'ye dönüştürür. Her tablo için tekrarlayan CRUD kodları yazmayı bırakın.

## Özellikler

- **Otomatik CRUD**: Tüm tablolar için GET, POST, PATCH, DELETE endpoint'leri
- **Gelişmiş Filtreleme**: `eq`, `gt`, `like`, `in` gibi operatörler
- **Sayfalama**: `limit` ve `offset` desteği
- **Sıralama**: `order` parametresi ile kolon bazlı sıralama
- **Güvenlik**: JWT kimlik doğrulama, tablo erişim kontrolü
- **Performans**: Go ile yazılmış, Redis önbellekleme
- **SQL Injection Koruması**: Tüm sorgular prepared statement kullanır
- **Case-Insensitive**: Tablo ve kolon isimlerinde büyük/küçük harf duyarsız
- **Identifier Escaping**: MySQL reserved words ve özel karakterler için otomatik koruma
- **Docker Ready**: Tek komut ile dağıtım

## Hızlı Başlangıç

```bash
# Bağımlılıkları indir
go mod download

# Veritabanı bilgilerinizle config/config.yaml dosyasını düzenleyin

# API'yi çalıştırın
go run cmd/instantgate/main.go -config config/config.yaml

# Veya derleyip çalıştırın
go build -o bin/instantgate.exe cmd/instantgate/main.go
.\bin\instantgate.exe -config config\config.yaml
```

API `http://localhost:8080` adresinde çalışır

### Docker

```bash
docker-compose -f deployments/docker-compose.yml up
```

## API Endpoint'leri

```bash
# Sağlık kontrolü
curl http://localhost:8080/health

# Tüm tabloları listele
curl http://localhost:8080/api/schema

# Tablo şemasını getir
curl http://localhost:8080/api/schema/:table

# Tablo verilerini getir (:table yerine gerçek tablo adı)
curl http://localhost:8080/api/:table

# Filtrelerle
curl "http://localhost:8080/api/:table?status=active&age=gt.18"

# Sayfalama ile
curl "http://localhost:8080/api/:table?limit=10&offset=20"

# Sıralama ile
curl "http://localhost:8080/api/:table?order=created_at.desc"

# Tekil kayıt getir
curl http://localhost:8080/api/:table/:id

# Kayıt oluştur
curl -X POST http://localhost:8080/api/:table \
  -H "Content-Type: application/json" \
  -d '{"field":"value"}'

# Kayıt güncelle
curl -X PATCH http://localhost:8080/api/:table/:id \
  -H "Content-Type: application/json" \
  -d '{"field":"newvalue"}'

# Kayıt sil
curl -X DELETE http://localhost:8080/api/:table/:id
```

## Filtre Operatörleri

| Operatör | Açıklama | Örnek |
|----------|----------|-------|
| `eq` | Eşit | `?status=active` |
| `ne` | Eşit değil | `?status=ne.inactive` |
| `gt` | Büyük | `?age=gt.18` |
| `gte` | Büyük veya eşit | `?age=gte.18` |
| `lt` | Küçük | `?price=lt.100` |
| `lte` | Küçük veya eşit | `?price=lte.100` |
| `like` | LIKE desen | `?name=like.%john%` |
| `in` | IN listesi | `?status=in.active,pending` |
| `nin` | NOT IN listesi | `?status=nin.deleted` |

## Yapılandırma

`config/config.yaml` dosyasını düzenleyin:

```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s

database:
  driver: mysql          # mysql veya postgres
  host: localhost
  port: 3306
  name: mydb
  user: root
  password: ""
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

jwt:
  secret: productionda-degistirin
  expiry: 24h

security:
  enabled: true
  require_auth: false   # true yaparsanız tüm endpoint'ler JWT ister
  whitelist: []         # Boş = tüm tablolara izin ver
  blacklist: []         # Engellenecek tablolar

redis:
  host: localhost
  port: 6379
  cache_ttl: 5m
```

## Güvenlik

### Tablo Erişim Kontrolü

Sadece belirli tablolara izin ver:
```yaml
security:
  whitelist: ["users", "products", "orders"]
```

Belirli tabloları engelle:
```yaml
security:
  blacklist: ["admin_users", "secrets", "config"]
```

### SQL Injection Koruması

Tüm sorgular prepared statements kullanır. Kullanıcı girdisi hiçbir zaman SQL'e concat edilmez. Ayrıca tüm tablo ve kolon isimleri otomatik olarak backtick ile escape edilir.

## Test Arayüzü

`test.html` dosyasını tarayıcınızda açarak tüm endpoint'leri keşfedebilir ve test edebilirsiniz.

## Proje Yapısı

```
cmd/instantgate/main.go          # Giriş noktası
internal/
  api/                            # HTTP router, handler'lar, middleware
  database/mysql/                 # MySQL sürücüsü, introspection
  query/                          # SQL builder, filtreler
  cache/                          # Redis önbellekleme
  security/                       # JWT, erişim kontrolü
config/config.yaml                # Yapılandırma
test.html                         # API test arayüzü
```

## Geliştirmeler

- **Case-Insensitive Lookup**: Tablo ve kolon isimleri büyük/küçük harf duyarsızdır
- **NULL Değer Handling**: NULL değerler düzgün şekilde JSON'a dönüştürülür
- **Time Format**: `time.Time` tipleri RFC3339 formatında JSON'a serile edilir
- **Error Logging**: Tüm hatalar konsola loglanır, debug kolaylığı sağlanır
- **Identifier Escaping**: Reserved words (`lock`, `key`, `order` vb.) içeren kolonlar otomatik korunur

## Lisans

MIT License