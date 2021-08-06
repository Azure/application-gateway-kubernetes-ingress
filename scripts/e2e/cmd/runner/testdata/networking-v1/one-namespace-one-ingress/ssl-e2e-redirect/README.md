To add the root certificate to app gateway, use

```
az network application-gateway root-cert create -n test --cert-file test.crt --gateway-name <gateway> --resource-group <resgp>
```

To generate a new self-signed certificate:
```
openssl ecparam -out test.key -name prime256v1 -genkey
openssl req -new -sha256 -key test.key -out test.csr -subj "/CN=test"
openssl x509 -req -sha256 -days 720 -in test.csr -signkey test.key -out test.crt
```

If you are using a different certificate, don't forget to update the tls secret in the app.yaml.
```
# set tls.crt with:
cat test.crt | base64 -w0

# set tls.key with:
cat test.key | base64 -w0
```