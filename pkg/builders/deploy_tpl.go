package builders

const cmtpl=`
  dbConfig:
    dsn: "[[ .Dsn ]]"
    maxOpenConn: [[ .MaxOpenConn ]]
    maxLifeTime: [[ .MaxLifeTime ]]
    maxIdleConn: [[ .MaxIdleConn ]]
  appConfig:
    rpcPort: 8081
    httpPort: 8090
  apis:
    - name: test
      sql: "select * from test"
`

// md5 设置在template 的 annotation中，所以当annotation发生变化时，所有的pod 信息都会进行更新为你声明的内容 ，从而更新了/app/app.yml
const deployTpl = `
apiVersion: apps/v1
kind: Deployment
metadata:   
  name: dbcore-{{ .Name }}
  namespace: {{ .Namespace}}
spec:
  selector:
    matchLabels:
      app: dbcore-{{ .Namespace}}-{{ .Name }}
  replicas: 1
  template:
    metadata:
      annotations:
        dbcore.config/md5: ''
      labels:
        app: dbcore-{{ .Namespace}}-{{ .Name }}
        version: v1
    spec:
      containers:
        - name: dbcore-{{ .Namespace}}-{{ .Name }}-container
          image: docker.io/shenyisyn/dbcore:v1
          imagePullPolicy: IfNotPresent
          volumeMounts:
            - name: configdata
              mountPath: /app/app.yml
              subPath: app.yml
          ports:
            - containerPort: 8081
            - containerPort: 8090
      volumes:
        - name: configdata
          configMap:
            defaultMode: 0644
            name: dbcore-{{ .Name }}
`
