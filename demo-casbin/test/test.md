


```bash
curl -H "X-User: alice" http://localhost:8080/api/user

## → 200 OK
GET OK
X-User: alice
```



```bash
# alice 是 admin，应能访问 /api/user
curl -H "X-User: alice" http://localhost:8080/api/user
# → 200 OK
```

```bash
# bob 是普通用户，访问 /api/user 应被拒绝
curl -H "X-User: bob" http://localhost:8080/api/user
# → 403 Forbidden
```

```bash
# bob 可以访问自己的 profile
curl -H "X-User: bob" http://localhost:8080/api/profile
# → 200 OK
```


