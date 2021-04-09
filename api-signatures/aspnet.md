# ASP.NET

## Signatures

GET /Help

```
API Help Page</title>
```

OR

```
My ASP.NET Application</p>
```

OR

```
<td class="api-documentation">
```

GET /thisdoesnotexist

```
<Message>No HTTP resource was found that matches the request URI
```

## Additional References

- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=%22No+HTTP+resource+was+found+that+matches+the+request+URI%22
- https://censys.io/ipv4/report?field=443.https.tls.certificate.parsed.subject.organization.raw&max_buckets=1000&q=%22My+ASP.NET+Application%3C%2Fp%3E%22