## Aider-Proxy

TabbyAPI can't load & server more than one model at the same time. It loads/unloads based on request. Very time-consuming.
I don't want to dig into TabbyAPI's code and add patches to it right now.
I just want to use Aider's architect mode with TabbyAPI. So I wrote an extremely simple reverse proxy for it.


### Build and Run                                  
To build the reverse proxy, ensure you have Go installed, then run:
```bash
 go build
```

This will generate an executable in the current directory. You can run it with:
./<aider-proxy>


### Configuration                                  

Configuration is done via a config.yml file. You can create this file by copying the config.example.yml provided in the repository and modifying it to suit your setup.

#### Example config.yml                               
```yaml
proxy:
  listen_port: ":5003"

servers:
  - name: "Tabby:QwQ-32B"
    url: "http://127.0.0.1:5001"
  - name: "Tabby:Qwen-2.5-Coder-32B"
    url: "http://127.0.0.1:5002"

routing:
  rules:
    - model: "QwQ-32B"
      server: "Tabby:QwQ-32B"
    - model: "Q25_32B-coder-5bpw"
      server: "Tabby:Qwen-2.5-Coder-32B"
  default_server: "Tabby:QwQ-32B"
```
- **proxy.listen_port**: The port on which the proxy will listen for incoming requests. Default is ":5003".                                                
- **servers**: A list of TabbyAPI servers you want to route requests to. Each server has a name and a url.                                                 
- **routing.rules**: Rules that map model names to server names. When a request specifies a model, the proxy routes it to the corresponding server.          
- **default_server**: The server to use if no routing rule matches the request's model.

#### Usage                                      
-  **Start the Proxy**: Run the built executable. It will start listening on the port specified in config.yml.                                                
- **Send Requests**: Send JSON requests to the proxy's listen port. The proxy will forward these requests to the appropriate TabbyAPI server based on the "model" field in the request body.                                           
Example request body, I guess:
```json
{
  "model": "QwQ-32B",
  "prompt": "Hello, world!"
}   
```                                                                        
#### Notes                                      

- This proxy only supports JSON requests with a "model" field in the request body.                                                                        
- Ensure that the TabbyAPI servers are running and accessible at the URLs specified in config.yml.                                                     
- This is a basic reverse proxy setup without advanced features like load balancing or request queuing.                                                
- Disclaimer: Use at your own risk. This was a quick late-night project to solve a specific problem that I didn't want to bother with 3rd party software. Feel free to improve and contribute back.

#### Contributing                                  
If you find issues or have suggestions for improvements, feel free to open an issue or submit a pull request.