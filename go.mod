module github.com/hpe-storage/common-host-libs

go 1.16

require (
	github.com/BurntSushi/toml v0.4.1 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.1.1 // indirect
	github.com/Scalingo/go-etcd-lock v3.0.1+incompatible
	github.com/coreos/bbolt v0.0.0-00010101000000-000000000000 // indirect
	github.com/coreos/etcd v3.3.13+incompatible
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-ole/go-ole v1.2.4
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/gorilla/context v1.1.2-0.20190627024605-8559d4a6b87e // indirect
	github.com/gorilla/mux v1.6.2
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/hectane/go-acl v0.0.0-20190523051433-dfeb47f3e2ef
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/opentracing/opentracing-go v1.2.0
	github.com/prometheus/client_golang v1.11.0 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/sparrc/go-ping v0.0.0-20190613174326-4e5b6552494c
	github.com/stretchr/testify v1.7.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802 // indirect
	github.com/uber/jaeger-client-go v2.29.1+incompatible
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	go.uber.org/atomic v1.8.1-0.20210622073649-557b938325dc // indirect
	go.uber.org/zap v1.19.0 // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/sys v0.0.0-20210603081109-ebe580a85c40
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/grpc v1.33.1
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/errgo.v1 v1.0.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.5

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0
