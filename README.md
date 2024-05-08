# HuggingChatAPI
逆向HuggingChat，并提供Openai兼容的API接口

进入[HuggingChat官网](https://huggingface.co/chat)并登录后，取出cookie中hf-chat的值作为Authorization

由于调用创建会话的接口创建出的会话总是莫名呈不可用状态，所以需要提前为每个Model创建好会话