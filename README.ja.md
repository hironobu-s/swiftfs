# ObjFS

ObjFSは各種オブジェクトストレージをFUSE(Filesystem in Userspace)を使用しマウントするファイルシステムです。現在OpenStack Swiftのみ対応しています。基本的にLinux系システムとMacFUSEを想定しています。

## インストール

以下のコマンドを実行することで、カレントディレクトリにobjcsコマンドがインストールされます。他のパスにインストールする場合は、冒頭の変数を書き換えて下さい。

### Linux(amd64)

```shell
F=objfs curl -sL https://github.com/hironobu-s/objfs/releases/download/current/objfs.amd64.gz | zcat > $F && chmod +x $F
```

### Max OSX

```
F=objfs curl -sL https://github.com/hironobu-s/objfs/releases/download/current/objfs-osx.amd64.gz | zcat > $F && chmod +x $F
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

objfsコマンドにコンテナ名マウントポイントを指定します。

```shell
$ objfs CONTAINER-NAME MOUNTPOINT
```

### アンマウント

fusermountコマンドを使用します。

```shell
$ fusermount -u MOUNTPOINT
```

### オプション

**--debug**

デバッグ出力をONにします

**--no-daemon**

objfsコマンドをフォアグラウンドで実行します。デバッグ用です。

**--logfile, -l**

指定したファイルにデバッグ情報などが書き込まれます。

**--create-container, -c**

コマンドライン引数で指定されたコンテナが存在しなかった場合にコンテナを作成します。このオプションを指定しない場合、コンテナが存在しない場合エラーになります。

## LISENCE

BSD License
