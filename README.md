# HuggingChatAPI

逆向 HuggingChat，并提供 OpenAI 兼容的 API 接口。

## 使用说明

1. **访问 HuggingChat 官网**  
   进入 [HuggingChat官网](https://huggingface.co/chat) 并登录。

2. **获取 Authorization**  
   从浏览器的 Cookie 中取出 `hf-chat` 的值，作为 Authorization。

3. **会话创建注意事项**  
   由于调用创建会话的接口创建出的会话总是呈不可用状态，因此需要提前为每个模型创建好会话。

## 调用说明

支持的接口：

- **获取模型列表**: `GET /v1/models`
- **聊天补全**: `POST /v1/chat/completions`

### 请求方法

您可以使用以下免费反代地址进行请求（国内可用，标准限制每天总请求上限为 10 万次，建议自行部署）：

`https://hf-api.464888.xyz`

在下列示例中，将使用免费地址，您可以根据需要替换为 `localhost:5695`（默认运行的端口）或您自行部署的地址。例如：

```bash
# 获取模型列表
curl -X GET "https://hf-api.464888.xyz/v1/models"

# 聊天补全
curl -X POST "https://hf-api.464888.xyz/v1/chat/completions" \
-H "Authorization: Bearer YOUR_AUTHORIZATION_TOKEN" \
-H "Content-Type: application/json" \
-d '{
  "model": "meta-llama/Meta-Llama-3.1-70B-Instruct",
  "messages": [{"role": "user", "content": "Hello!"}]
}'
```

如果您在本地运行服务，可以使用以下命令：

```bash
# 获取模型列表
curl -X GET "http://localhost:5695/v1/models"

# 聊天补全
curl -X POST "http://localhost:5695/v1/chat/completions" \
-H "Authorization: Bearer YOUR_AUTHORIZATION_TOKEN" \
-H "Content-Type: application/json" \
-d '{
  "model": "meta-llama/Meta-Llama-3.1-70B-Instruct",
  "messages": [{"role": "user", "content": "Hello!"}]
}'
```

### 支持的模型

- `meta-llama/Meta-Llama-3.1-70B-Instruct`
- `CohereForAI/c4ai-command-r-plus-08-2024`
- `Qwen/Qwen2.5-72B-Instruct`
- `nvidia/Llama-3.1-Nemotron-70B-Instruct-HF`
- `meta-llama/Llama-3.2-11B-Vision-Instruct`
- `NousResearch/Hermes-3-Llama-3.1-8B`
- `mistralai/Mistral-Nemo-Instruct-2407`

## 部署方案

### 部署要求

部署本项目需要支持海外环境访问 HuggingFace。如果您在国内部署，请确保服务器能够访问海外网络。

### 使用 Docker Compose

1. **创建目录并下载配置文件**

   ```bash
   mkdir HuggingChatAPI
   cd HuggingChatAPI
   curl -O https://github.com/kkkunny/HuggingChatAPI/docker-compose.yml
   ```

2. **启动服务**

   ```bash
   docker-compose up -d
   ```

### 使用 Docker 直接运行

如果不想使用 Docker Compose，可以直接运行以下命令：

```bash
docker run -d --name HuggingChat -p 5695:80 kkkunny/hugging-chat-api:latest
```

### 使用 Koyeb 一键部署

1. **准备工作**  
   - 确保您有一个 GitHub 账号，并提前登录 Koyeb（需要没有正在使用的免费计划的账号，因为免费计划不会休眠）。
   - Fork 本仓库。

2. **一键部署**  
   点击下面的图标进行一键部署，选择免费计划：

   [![Deploy to Koyeb](https://www.koyeb.com/static/images/deploy/button.svg)](https://app.koyeb.com/deploy?name=huggingchatapi&type=git&repository=2328760190%2FHuggingChatAPI&branch=master&builder=dockerfile&regions=was&env%5B%5D=&ports=80%3Bhttp%3B%2F)

3. **查看项目**  
   等待部署完成后，您可以在 [Koyeb 控制台](https://app.koyeb.com/) 中查看项目，点击项目即可查看链接（注意：国内可能有污染，若有域名可以使用 Cloudflare 反代一下）。

4. **Cloudflare 反代方法**  
   如果您希望使用 Cloudflare 反代来访问 HuggingChatAPI，特别是在 Koyeb 部署时，可以使用以下代码：

   ```javascript
   export default {
     async fetch(request, env) {
       // 创建目标 URL
       const url = new URL(request.url);
       url.hostname = '你的部署地址，不带开头https://，结尾不带/';
       
       // 创建新的请求对象
       const newRequest = new Request(url, {
         method: request.method,
         headers: request.headers,
         body: request.method === 'POST' ? request.body : null,
         redirect: 'follow'
       });

       // 转发请求并返回响应
       return fetch(newRequest);
     }
   }
   ```

   请将 `你的部署地址` 替换为您实际的部署地址。反代的目的是为了确保在 Koyeb 部署后能够顺利访问 HuggingChatAPI。
