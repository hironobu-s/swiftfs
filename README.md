# SwiftFS

[English Document](README.en.md)

SwiftFSはOpenStack Swift[^1]のコンテナをFUSE(Filesystem in Userspace)を使用しマウントするファイルシステムです。

[^1]: OpenStackの分散オブジェクトストレージシステム

## 動作確認環境

- ConoHaオブジェクトストレージ (https://www.conoha.jp/)
- Rackspace Cloud Files (http://www.rackspace.com/cloud/files)


## インストール

現状ではLinux(amd64)環境でのみ動作します。

以下のコマンドを実行することで、カレントディレクトリにswiftfsコマンドがインストールされます。

```shell
curl -sL https://github.com/hironobu-s/swiftfs/releases/download/current/swiftfs.amd64.gz | zcat > swiftfs && chmod +x swiftfs
```


## 使い方

### 認証情報の設定

まず、OpenStack APIへの認証情報を設定する必要があります。コマンドラインオプションと環境変数のどちらかで渡すことができます。

```shell
--os-user-id                 (OpenStack) User ID [$OS_USERID]
--os-username                (OpenStack) Username [$OS_USERNAME]
--os-password                (OpenStack) Password [$OS_PASSWORD]
--os-tenant-id               (OpenStack) Tenant Id [$OS_TENANT_ID]
--os-tenant-name             (OpenStack) Tenant Name [$OS_TENANT_NAME]
--os-auth-url                (OpenStack) Auth URL(required) [$OS_AUTH_URL]
--os-region-name             (OpenStack) Region Name [$OS_REGION_NAME]
```

### マウント

swiftfsコマンドにコンテナ名マウントポイントを指定します。

```shell
$ swiftfs CONTAINER-NAME MOUNTPOINT
```

### アンマウント

fusermountコマンドを使用します。

```shell
$ fusermount -u MOUNTPOINT
```

### オプション

swiftfsコマンドに-hオプションをつけて実行すると、オプションの一覧が表示されます。

**--debug**

デバッグ出力をONにします

**--no-daemon**

swiftfsコマンドをフォアグラウンドで実行します。デバッグ用です。

**--logfile, -l**

指定したファイルにデバッグ情報などが書き込まれます。

**----object-cache-time**

オブジェクトの一覧をキャッシュする秒数を設定します。オブジェクト一覧の取得にはAPIを実行する必要があり、キャッシュすることにより回数が減りパフォーマンスが向上します。ただし、複数ノードからswiftfsでマウントしている場合、キャッシュにより差異が出てしまうことがあります。デフォルト値は-1で、これはキャッシュしないことを意味します。

**--create-container, -c**

コマンドライン引数で指定されたコンテナが存在しなかった場合にコンテナを作成します。このオプションを指定しない場合、コンテナが存在しない場合エラーになります。

## やることリスト

- chmod/chownのサポート
- HTTP圧縮のサポート(net/httpパッケージが未サポート)
- ~~ObjectListをキャッシュしたい~~
- ~~マルチスレッドで書き込むとまれに正しく書き込めないことがある~~ (たぶん直った)
- オブジェクト数が増えた時のパフォーマンス確保
- その他バグフィクス

## License

MIT License

## Author

Hironobu Saitoh
<hiro@hironobu.org>
