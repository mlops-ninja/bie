ngrock for files


echo README.md | xargs -I{} curl -v -X POST -F "file=@{}" https://bie.mlops.ninja/upload/SNd594uNdqlVIVz06twS9FM674iUFFFY





# Security



## Local development environment 

1. Use [mkcert](https://github.com/FiloSottile/mkcert) to generate certificates for `bie.test` and `*.bie.test`:



1. Relay owner can not issue his own certificate, because the self-signed CA is not shared to him.

    **cURL** command is smart enough to **switch off** `ca-certs` if `--cacert` flag is provided. For example:

    ```bash
    $ curl --cacert <(echo '-----BEGIN CERTIFICATE-----
    MIICGTCCAYKgAwIBAgIBAjANBgkqhkiG9w0BAQsFADAQMQ4wDAYDVQQDEwVCaWVD
    QTAeFw0yNTAzMTExNTQ5NDVaFw0yNjAzMTExNTQ5NDVaMEsxSTBHBgNVBAMTQDAx
    LXRkNnV0anhmdWx5Y3B4Z3ZidmRmaGt3d2gyYXJqYWZlcmZ6ZHJsY3BiZDMzaWth
    bWUyZWEuYmllLnRlc3QwgZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBAMxoE0E9
    GjA9GI+OuLA3GO9JoEiDyfC9NEk8dqzR8+8bdG3jvRjWfEFG+0pmqQMDY6OR2gtr
    SnvVivHzDYARbZlLLBBg+xhH7DNzc2lop+y3iENcHnKOmrU/BuwbBt9vKBksdz9j
    JDmEn6I5WoRdR4njL4PG9KyCuWipBaI893UbAgMBAAGjSDBGMA4GA1UdDwEB/wQE
    AwIHgDATBgNVHSUEDDAKBggrBgEFBQcDATAfBgNVHSMEGDAWgBT7znfJcP6uqQrL
    CEMuzA1XDgUIejANBgkqhkiG9w0BAQsFAAOBgQBV0qIc7JhFHN1GbM+ggnqXM3yD
    5m2SWU5e+cNAqZEL1FRs17IcpAT2lu9k5RM4SjXiwnb0kRCWs9kf5mEP7zvQYjWU
    JY2miTGbzxF1drLMSm6LHke9DBrd5/ec4xbzmyRriFtZX7uisCGDlaB+RUrLcHBD
    5qPlSMvnChlnrOOWFw==
    -----END CERTIFICATE-----
    ') https://google.com
    ```

    will fail, even if your system has valid google's CA installed operation system wide.