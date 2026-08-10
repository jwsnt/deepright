package ai.open.right.workflow.mcp.client.impl;

import ai.open.right.WorkflowException;
import ai.open.right.listener.EventListenerService;
import ai.open.right.resouce.PlaceholderResolver;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.config.NamesService;
import ai.open.right.workflow.flow.WorkflowWatcher;
import ai.open.right.workflow.flow.llm.config.LLMFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderFunCall;
import ai.open.right.workflow.flow.llm.provider.ProviderRequestService;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.mcp.client.*;
import ai.open.right.workflow.mcp.client.dimension.McpDimension;
import ai.open.right.workflow.mcp.client.stdio.McpStdioClient;
import ai.open.right.workflow.mcp.client.stream.McpStreamClient;
import ai.open.right.workflow.mcp.client.utils.McpContentUtils;
import ai.open.right.workflow.mcp.client.utils.McpToolsUtils;
import ai.open.right.workflow.mcp.config.McpConfigInit;
import ai.open.right.workflow.notify.NotifierService;
import ai.open.right.workflow.sync.SyncConfig;
import ai.open.right.workflow.sync.SyncWorkflowTask;
import com.fasterxml.jackson.core.JsonParseException;
import com.google.common.collect.ImmutableMap;
import jakarta.annotation.PreDestroy;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.BooleanUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.lang3.reflect.MethodUtils;
import org.apache.commons.pool2.BasePooledObjectFactory;
import org.apache.commons.pool2.PooledObject;
import org.apache.commons.pool2.impl.DefaultPooledObject;
import org.apache.commons.pool2.impl.GenericObjectPool;
import org.apache.commons.pool2.impl.GenericObjectPoolConfig;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;

import java.util.*;
import java.util.concurrent.*;

@Slf4j
@Setter
@Getter
public class McpClientServiceImpl implements McpConfigInit, McpClientService {

    public static final List<ProviderFunCall> EMPTY_FUNCALL = Collections.unmodifiableList(new ArrayList<ProviderFunCall>());

    public static final List<History> EMPTY_HISTORY = Collections.unmodifiableList(new ArrayList<History>());

    public static final String TYPE_STREAM = "streamableHttp";

    public static final String TYPE_STDIO = "stdio";

    protected Map<String, McpConfig> clients = new ConcurrentHashMap<String, McpConfig>();

    protected PlaceholderResolver placeholderResolver;

    protected CloseableHttpAsyncClient httpClient;

    protected EventListenerService eventListener;

    protected ExecutorService executorService;

    protected NotifierService notifierService;

    protected NamesService namesService;

    protected Integer timeBetweenEvictionRunsMillis;

    // MCP的线程池数量
    protected Integer processor;

    // MCP的执行超时
    protected Integer timeout;

    // MCP的获取超时
    protected Integer borrow;

    @Override
    public void init(Map<String, Object> config) throws Exception {
        // 获取MCP Servers配置
        Map<String, Object> servers = Map.class.cast(config.get("mcpServers"));
        if (log.isInfoEnabled()) {
            log.info("Mcp servers={}", servers);
        }
        if (!CollectionUtils.isEmpty(servers)) {
            // 计算线程池数量
            this.processor = this.processor != null ? this.processor : Runtime.getRuntime().availableProcessors();
            for (String key : servers.keySet()) {
                // 获取配个服务的配置
                Map<String, Object> server = Map.class.cast(servers.get(key));
                if (log.isDebugEnabled()) {
                    log.debug("Mcp server{},config={}", key, server);
                }
                if (!CollectionUtils.isEmpty(server)) {
                    int processor = Integer.parseInt(MapUtils.getString(server, "processor", String.valueOf(this.processor)));
                    int timeout = Integer.parseInt(MapUtils.getString(server, "timeout", String.valueOf(this.timeout)));
                    GenericObjectPoolConfig<McpClient> clientConfig = new GenericObjectPoolConfig<McpClient>();
                    clientConfig.setTimeBetweenEvictionRunsMillis(this.timeBetweenEvictionRunsMillis);
                    clientConfig.setTestOnBorrow(true);
                    clientConfig.setTestOnReturn(true);
                    clientConfig.setTestWhileIdle(true);
                    clientConfig.setMaxTotal(processor);
                    clientConfig.setMaxIdle(processor);
                    clientConfig.setMinIdle(processor);
                    this.clients.put(key, McpConfig.builder()
                            .client(new GenericObjectPool<McpClient>(new McpClientFactory(this.httpClient, key, server), clientConfig))
                            .timeout(timeout)
                            .build());
                    if (log.isDebugEnabled()) {
                        log.debug("Mcp server={}, processor={}, timeout={}", key, processor, timeout);
                    }
                }
            }
        }
    }

    @PreDestroy
    public void destroy() {
        for (String key : this.clients.keySet()) {
            this.clients.get(key).getClient().close();
        }
    }

    public McpResult<List<Map<String, Object>>> toolsCall(String client, String name, Map<String, Object> arguments, McpDimension mcpDimension) throws Exception {
        Assert.hasText(client, "Mcp client name can not be empty");
        Assert.hasText(name, "Mcp name can not be empty");
        WorkflowWatcher watcher = WorkflowWatcher.builder().build();
        McpConfig mcpConfig = this.clients.get(client);
        if (log.isDebugEnabled()) {
            log.debug("Mcp tools call {}/{} took {} milliseconds to get config", client, name, watcher.getConsuming());
        }
        Assert.notNull(mcpConfig, "Mcp client config can not be empty");
        Future<Object> future = null;
        try {
            McpResult<List<Map<String, Object>>> result = new McpResult<List<Map<String, Object>>>();
            result.setResult(List.class.cast((future = this.executorService.submit(new McpFuture(mcpConfig.getClient(), this.borrow, "toolsCall", new Object[]{name, !CollectionUtils.isEmpty(arguments) ? arguments : new HashMap<String, Object>(), mcpDimension}))).get(mcpConfig.getTimeout(), TimeUnit.MILLISECONDS)));
            result.setClient(client);
            result.setName(name);
            if (log.isDebugEnabled()) {
                log.debug("Mcp tools call {}/{} took {} milliseconds to get result", client, name, watcher.getConsuming());
            }
            if (this.eventListener != null) {
                this.eventListener.listen(new McpEvent(mcpDimension, result, client, name));
            }
            return result;
        } catch (TimeoutException e) {
            future.cancel(true);
            throw e;
        }
    }

    public McpResult<List<Map<String, Object>>> toolsCall(String name, Map<String, Object> arguments, McpDimension mcpDimension) throws Exception {
        Assert.hasText(name, "Mcp client name can not be empty");
        // 对名称进行解码（BIZ@WORKFLOW）
        String[] pair = this.namesService.decode(name);
        Assert.isTrue(pair.length == 2, "Mcp name is invalid");
        return this.toolsCall(pair[0], pair[1], arguments, mcpDimension);
    }

    public List<ProviderFunCall> toolsList(String client, McpDimension mcpDimension) throws Exception {
        Assert.hasText(client, "Mcp client name can not be empty");
        WorkflowWatcher watcher = WorkflowWatcher.builder().build();
        McpConfig mcpConfig = this.clients.get(client);
        if (log.isDebugEnabled()) {
            log.debug("Mcp tools list {} took {} milliseconds to get config", client, watcher.getConsuming());
        }
        Assert.notNull(mcpConfig, "Mcp client config can not be empty");
        Future<Object> future = null;
        try {
            List<Map<String, Object>> response = List.class.cast((future = this.executorService.submit(new McpFuture(mcpConfig.getClient(), this.borrow, "toolsList", new Object[]{mcpDimension}))).get(mcpConfig.getTimeout(), TimeUnit.MILLISECONDS));
            if (CollectionUtils.isEmpty(response)) {
                if (log.isDebugEnabled()) {
                    log.debug("Mcp tools list {} took {} milliseconds to get empty result", client, watcher.getConsuming());
                }
                // 内部事件监听并返回
                if (this.eventListener != null) {
                    this.eventListener.listen(new McpEvent(mcpDimension, McpClientServiceImpl.EMPTY_FUNCALL, client));
                }
                return McpClientServiceImpl.EMPTY_FUNCALL;
            }
            // 构建Fun Call
            List<ProviderFunCall> funCalls = new ArrayList<ProviderFunCall>();
            for (Map<String, Object> data : response) {
                try {
                    LLMFunCall funCall = new LLMFunCall();
                    String name = String.class.cast(data.get("name"));
                    Assert.hasText(name, "Mcp tools name can not be empty");
                    // 对名称进行编码
                    funCall.setName(this.namesService.encode(NamesService.PREFIX_TOOLS, client, name));
                    funCall.setDescription(String.class.cast(data.get("description")));
                    Map<String, Object> inputSchema = Map.class.cast(data.get("inputSchema"));
                    if (!CollectionUtils.isEmpty(inputSchema)) {
                        // 过滤无法用于FunCall的Properties
                        funCall.setProperties(McpToolsUtils.filter(Map.class.cast(inputSchema.get("properties"))));
                        funCall.setRequired(List.class.cast(inputSchema.get("required")));
                    }
                    funCalls.add(funCall);
                } catch (Exception e) {
                    WorkflowException.dolog(e);
                }
            }
            if (log.isDebugEnabled()) {
                log.debug("Mcp tools list {} took {} milliseconds to get result({})", client, watcher.getConsuming(), funCalls.size());
            }
            if (this.eventListener != null) {
                // 内部事件监听并返回
                this.eventListener.listen(new McpEvent(mcpDimension, funCalls, client));
            }
            return funCalls;
        } catch (TimeoutException e) {
            future.cancel(true);
            throw e;
        }
    }

    public McpResult<List<History>> promptGet(String client, String name, Map<String, Object> arguments, McpRuntime mcpRuntime, McpDimension mcpDimension) throws Exception {
        Assert.hasText(client, "Mcp client name can not be empty");
        WorkflowWatcher watcher = WorkflowWatcher.builder().build();
        McpConfig mcpConfig = this.clients.get(client);
        if (log.isDebugEnabled()) {
            log.debug("Mcp prompt get {} took {} milliseconds to get config", client, watcher.getConsuming());
        }
        Assert.notNull(mcpConfig, "Mcp client config can not be empty: " + client);
        if (!StringUtils.isEmpty(name)) {
            Future<Object> future = null;
            try {
                // 指定Name的调用
                Map<String, Object> response = Map.class.cast((future = this.executorService.submit(new McpFuture(mcpConfig.getClient(), this.borrow, "promptGet", new Object[]{name, !CollectionUtils.isEmpty(arguments) ? arguments : new HashMap<String, Object>(), mcpDimension}))).get(mcpConfig.getTimeout(), TimeUnit.MILLISECONDS));
                if (CollectionUtils.isEmpty(response)) {
                    // 没有结果
                    McpResult<List<History>> result = new McpResult<List<History>>();
                    result.setResult(McpClientServiceImpl.EMPTY_HISTORY);
                    result.setClient(client);
                    result.setName(name);
                    if (log.isDebugEnabled()) {
                        log.debug("Mcp prompt get {} took {} milliseconds to get empty result", client, watcher.getConsuming());
                    }
                    if (this.eventListener != null) {
                        // 内部事件监听并返回
                        this.eventListener.listen(new McpEvent(mcpDimension, result, client, name));
                    }
                    return result;
                }
                List<Map<String, Object>> messages = List.class.cast(response.get("messages"));
                Assert.notEmpty(messages, "Mcp prompt message can not be empty");
                McpResult<List<History>> result = new McpResult<List<History>>();
                // 构建为记忆
                result.setResult(this.buildHistories(messages));
                result.setClient(client);
                result.setName(name);
                if (log.isDebugEnabled()) {
                    log.debug("Mcp prompt get {} took {} milliseconds to get result", client, watcher.getConsuming());
                }
                if (this.eventListener != null) {
                    // 内部事件监听并返回
                    this.eventListener.listen(new McpEvent(mcpDimension, result, client, name));
                }
                return result;
            } catch (TimeoutException e) {
                future.cancel(true);
                throw e;
            }
        } else {
            // 不指定名称，Fun Call查找
            if (log.isDebugEnabled()) {
                log.debug("Mcp prompt get runtime={}-{}", name, mcpRuntime);
            }
            Assert.notNull(mcpRuntime, "Mcp prompt get runtime can not be empty: " + client);
            StringBuffer query = new StringBuffer(mcpRuntime.getPrefix()).append(mcpRuntime.getWorkTask().getQuery()).append(mcpRuntime.getSuffix());
            if (log.isDebugEnabled()) {
                log.debug("Mcp prompt get reQuery={}-{}", name, query);
            }
            SyncConfig syncConfig = SyncConfig.builder()
                    // 构建特殊查询Metadata
                    .metadata(ImmutableMap.of(ProviderRequestService.KEY_FUN_INTERNAL, this.promptList(client, mcpDimension)))
                    .workflow(ProviderRequestService.KEY_FUN_SELECT)
                    .timeout(mcpRuntime.getTimeout(this.timeout))
                    .workTask(mcpRuntime.getWorkTask())
                    .reQuery(query.toString())
                    .build();
            // Fun Call模型选择
            String response = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
            if (log.isDebugEnabled()) {
                log.debug("Mcp prompt get (fun call) original response={}-{}", name, response);
            }
            try {
                if (!StringUtils.isEmpty(response)) {
                    McpResult<List<History>> result = new McpResult<List<History>>();
                    result.setResult(Arrays.asList(JsonUtils.read(response, History[].class)));
                    result.setClient(client);
                    result.setName(name);
                    if (log.isDebugEnabled()) {
                        log.debug("Mcp prompt get {} took {} milliseconds to get fun call result", client, watcher.getConsuming());
                    }
                    if (this.eventListener != null) {
                        // 内部事件监听并返回
                        this.eventListener.listen(new McpEvent(mcpDimension, result, client, name));
                    }
                    return result;
                } else {
                    McpResult<List<History>> result = new McpResult<List<History>>();
                    result.setResult(McpClientServiceImpl.EMPTY_HISTORY);
                    result.setClient(client);
                    result.setName(name);
                    if (log.isDebugEnabled()) {
                        log.debug("Get mcp prompt get {} took {} milliseconds to get fun call empty result", client, watcher.getConsuming());
                    }
                    if (this.eventListener != null) {
                        // 内部事件监听并返回
                        this.eventListener.listen(new McpEvent(mcpDimension, result, client, name));
                    }
                    return result;
                }
            } catch (JsonParseException e) {
                if (log.isDebugEnabled()) {
                    // 如果无匹配Prompt时Json会无法解析
                    log.debug(e.getMessage(), e);
                }
                McpResult<List<History>> result = new McpResult<List<History>>();
                result.setResult(McpClientServiceImpl.EMPTY_HISTORY);
                result.setClient(client);
                result.setName(name);
                if (log.isDebugEnabled()) {
                    log.debug("Get mcp prompt get {} took {} milliseconds to get fun call empty result", client, watcher.getConsuming());
                }
                if (this.eventListener != null) {
                    // 内部事件监听并返回
                    this.eventListener.listen(new McpEvent(mcpDimension, result, client, name));
                }
                return result;
            }
        }
    }

    @Override
    public McpResult<List<History>> promptGet(String client, String name, Map<String, Object> arguments, McpDimension mcpDimension) throws Exception {
        return this.promptGet(client, name, arguments, null, mcpDimension);
    }

    public McpResult<List<History>> promptGet(String name, Map<String, Object> arguments, McpDimension mcpDimension) throws Exception {
        Assert.hasText(name, "Mcp client name can not be empty");
        String[] pair = this.namesService.decode(name);
        return this.promptGet(pair[0], pair[1], arguments, null, mcpDimension);
    }

    public List<ProviderFunCall> promptList(String client, McpDimension mcpDimension) throws Exception {
        Assert.hasText(client, "Mcp client name can not be empty");
        WorkflowWatcher watcher = WorkflowWatcher.builder().build();
        McpConfig mcpConfig = this.clients.get(client);
        if (log.isDebugEnabled()) {
            log.debug("Mcp prompt list {} took {} milliseconds to get config", client, watcher.getConsuming());
        }
        Assert.notNull(mcpConfig, "Mcp client config can not be empty: " + client);
        Future<Object> future = null;
        try {
            List<Map<String, Object>> response = List.class.cast((future = this.executorService.submit(new McpFuture(mcpConfig.getClient(), this.borrow, "promptList", new Object[]{mcpDimension}))).get(mcpConfig.getTimeout(), TimeUnit.MILLISECONDS));
            if (CollectionUtils.isEmpty(response)) {
                if (log.isDebugEnabled()) {
                    log.debug("Mcp prompt list {} took {} milliseconds to get empty result", client, watcher.getConsuming());
                }
                if (this.eventListener != null) {
                    // 内部事件监听并返回
                    this.eventListener.listen(new McpEvent(mcpDimension, McpClientServiceImpl.EMPTY_FUNCALL, client));
                }
                return McpClientServiceImpl.EMPTY_FUNCALL;
            }
            // 构建Fun Call
            List<ProviderFunCall> funCalls = new ArrayList<ProviderFunCall>();
            for (Map<String, Object> data : response) {
                try {
                    LLMFunCall funCall = new LLMFunCall();
                    String name = String.class.cast(data.get("name"));
                    Assert.hasText(name, "Mcp prompt name can not be empty");
                    // Name encode
                    funCall.setName(this.namesService.encode(NamesService.PREFIX_PROMPT, client, name));
                    funCall.setDescription(String.class.cast(data.get("description")));
                    List<Map<String, Object>> arguments = List.class.cast(data.get("arguments"));
                    Map<String, Object> properties = new HashMap<String, Object>();
                    List<String> required = new ArrayList<String>();
                    for (Map<String, Object> argument : arguments) {
                        String argumentDesc = String.class.cast(argument.get("description"));
                        String argumentName = String.class.cast(argument.get("name"));
                        if (BooleanUtils.isTrue(Boolean.class.cast(argument.get("required")))) {
                            required.add(argumentName);
                        }
                        properties.put(argumentName, ImmutableMap.<String, String>builder().put("description", StringUtils.defaultString(argumentDesc, "")).put("type", "string").build());
                    }
                    funCall.setProperties(properties);
                    funCall.setRequired(required);
                    funCalls.add(funCall);
                } catch (Exception e) {
                    WorkflowException.dolog(e);
                }
            }
            if (log.isDebugEnabled()) {
                log.debug("Mcp prompt list {} took {} milliseconds to get result", client, watcher.getConsuming());
            }
            if (this.eventListener != null) {
                // 内部事件监听并返回
                this.eventListener.listen(new McpEvent(mcpDimension, funCalls, client));
            }
            return funCalls;
        } catch (TimeoutException e) {
            future.cancel(true);
            throw e;
        }
    }

    public List<ProviderFunCall> resourcesTemplatesList(String client, McpDimension mcpDimension) throws Exception {
        Assert.hasText(client, "Mcp client name can not be empty");
        WorkflowWatcher watcher = WorkflowWatcher.builder().build();
        McpConfig mcpConfig = this.clients.get(client);
        if (log.isDebugEnabled()) {
            log.debug("Mcp resources templates list {} took {} milliseconds to get config", client, watcher.getConsuming());
        }
        Assert.notNull(mcpConfig, "Mcp client config can not be empty");
        Future<Object> future = null;
        try {
            List<Map<String, Object>> result = List.class.cast((future = this.executorService.submit(new McpFuture(mcpConfig.getClient(), this.borrow, "resourcesTemplatesList", new Object[]{mcpDimension}))).get(mcpConfig.getTimeout(), TimeUnit.MILLISECONDS));
            if (CollectionUtils.isEmpty(result)) {
                if (log.isDebugEnabled()) {
                    log.debug("Mcp resources templates list {} took {} milliseconds to get empty result", client, watcher.getConsuming());
                }
                if (this.eventListener != null) {
                    // 内部事件监听并返回
                    this.eventListener.listen(new McpEvent(mcpDimension, McpClientServiceImpl.EMPTY_FUNCALL, client));
                }
                return McpClientServiceImpl.EMPTY_FUNCALL;
            }
            // 构建Fun Call
            List<ProviderFunCall> funCalls = new ArrayList<ProviderFunCall>();
            for (Map<String, Object> resource : result) {
                try {
                    LLMFunCall each = new LLMFunCall();
                    String description = String.class.cast(resource.get("description"));
                    String uri = String.class.cast(resource.get("uriTemplate"));
                    String name = String.class.cast(resource.get("name"));
                    Assert.hasText(name, "Mcp resources templates list name can not be empty");
                    Assert.hasText(uri, "Mcp resources templates list  uri can not be empty");
                    // Optional description
                    each.setDescription("URI format: " + uri + ". " + StringUtils.defaultString(description, ""));
                    // Name编码
                    each.setName(this.namesService.encode(NamesService.PREFIX_RESOURCE, client, uri));
                    Map<String, Object> properties = new HashMap<String, Object>();
                    properties.put("uri", ImmutableMap.<String, String>builder().put("description", each.getDescription()).put("type", "string").build());
                    each.setRequired(Arrays.asList("uri"));
                    each.setProperties(properties);
                    funCalls.add(each);
                } catch (Exception e) {
                    WorkflowException.dolog(e);
                }
            }
            if (log.isDebugEnabled()) {
                log.debug("Mcp resources templates list {} took {} milliseconds to get result", client, watcher.getConsuming());
            }
            if (this.eventListener != null) {
                // 内部事件监听并返回
                this.eventListener.listen(new McpEvent(mcpDimension, funCalls, client));
            }
            return funCalls;
        } catch (TimeoutException e) {
            future.cancel(true);
            throw e;
        }
    }

    public List<ProviderFunCall> resourcesList(String client, McpDimension mcpDimension) throws Exception {
        Assert.hasText(client, "Mcp client name can not be empty");
        WorkflowWatcher watcher = WorkflowWatcher.builder().build();
        McpConfig mcpConfig = this.clients.get(client);
        if (log.isDebugEnabled()) {
            log.debug("Mcp resources list {} took {} milliseconds to get config", client, watcher.getConsuming());
        }
        Assert.notNull(mcpConfig, "Mcp client config can not be empty");
        Future<Object> future = null;
        try {
            List<Map<String, Object>> result = List.class.cast((future = this.executorService.submit(new McpFuture(mcpConfig.getClient(), this.borrow, "resourcesList", new Object[]{mcpDimension}))).get(mcpConfig.getTimeout(), TimeUnit.MILLISECONDS));
            if (CollectionUtils.isEmpty(result)) {
                if (log.isDebugEnabled()) {
                    log.debug("Mcp resources list {} took {} milliseconds to get empty result", client, watcher.getConsuming());
                }
                if (this.eventListener != null) {
                    // 内部事件监听并返回
                    this.eventListener.listen(new McpEvent(mcpDimension, McpClientServiceImpl.EMPTY_FUNCALL, client));
                }
                return McpClientServiceImpl.EMPTY_FUNCALL;
            }
            // 构建Fun Call
            List<ProviderFunCall> funCalls = new ArrayList<ProviderFunCall>();
            for (Map<String, Object> resource : result) {
                try {
                    LLMFunCall each = new LLMFunCall();
                    String description = String.class.cast(resource.get("description"));
                    String name = String.class.cast(resource.get("name"));
                    String uri = String.class.cast(resource.get("uri"));
                    Assert.hasText(name, "Mcp resources list name can not be empty");
                    Assert.hasText(uri, "Mcp resources list uri can not be empty");
                    // Optional description
                    each.setDescription("URI format: " + uri + ". " + StringUtils.defaultString(description, name));
                    // Name编码
                    each.setName(this.namesService.encode(NamesService.PREFIX_RESOURCE, client, uri));
                    Map<String, Object> properties = new HashMap<String, Object>();
                    properties.put("uri", ImmutableMap.<String, String>builder().put("description", each.getDescription()).put("type", "string").build());
                    each.setRequired(Arrays.asList("uri"));
                    each.setProperties(properties);
                    funCalls.add(each);
                } catch (Exception e) {
                    WorkflowException.dolog(e);
                }
            }
            if (log.isDebugEnabled()) {
                log.debug("Mcp resources list {} took {} milliseconds to get result", client, watcher.getConsuming());
            }
            if (this.eventListener != null) {
                // 内部事件监听并返回
                this.eventListener.listen(new McpEvent(mcpDimension, funCalls, client));
            }
            return funCalls;
        } catch (TimeoutException e) {
            future.cancel(true);
            throw e;
        }
    }

    public McpResult<String> resourcesRead(String client, String uri, McpRuntime mcpRuntime, McpDimension mcpDimension) throws Exception {
        Assert.hasText(client, "Mcp client name can not be empty");
        WorkflowWatcher watcher = WorkflowWatcher.builder().build();
        McpConfig mcpConfig = this.clients.get(client);
        if (log.isDebugEnabled()) {
            log.debug("Mcp resources read {} took {} milliseconds to get config", client, watcher.getConsuming());
        }
        Assert.notNull(mcpConfig, "Mcp client config can not be empty: " + client);
        if (!StringUtils.isEmpty(uri)) {
            Future<Object> future = null;
            try {
                // 指定URI
                List<Map<String, Object>> response = List.class.cast((future = this.executorService.submit(new McpFuture(mcpConfig.getClient(), this.borrow, "resourcesRead", new Object[]{uri, mcpDimension}))).get(mcpConfig.getTimeout(), TimeUnit.MILLISECONDS));
                if (CollectionUtils.isEmpty(response)) {
                    McpResult<String> empty = new McpResult<String>();
                    empty.setClient(client);
                    empty.setResult("");
                    empty.setName(uri);
                    if (log.isDebugEnabled()) {
                        log.debug("Mcp resources read {} took {} milliseconds to get empty result", client, watcher.getConsuming());
                    }
                    if (this.eventListener != null) {
                        // 内部事件监听并返回
                        this.eventListener.listen(new McpEvent(mcpDimension, empty, client, uri));
                    }
                    return empty;
                }
                StringBuffer content = new StringBuffer();
                for (Map<String, Object> resource : response) {
                    String type = String.class.cast(resource.get("mimeType"));
                    Assert.hasText(type, "Mcp resource mime type can not be empty");
                    String text = McpContentUtils.resource(type, resource);
                    // Not null
                    Assert.notNull(text, "Mcp resources text can not be empty");
                    content.append(text).append(System.lineSeparator());
                }
                McpResult<String> result = new McpResult<String>();
                result.setResult(content.toString());
                result.setClient(client);
                result.setName(uri);
                if (log.isDebugEnabled()) {
                    log.debug("Mcp resources read {} took {} milliseconds to get result", client, watcher.getConsuming());
                }
                if (this.eventListener != null) {
                    // 内部事件监听并返回
                    this.eventListener.listen(new McpEvent(mcpDimension, result, client, uri));
                }
                return result;
            } catch (TimeoutException e) {
                future.cancel(true);
                throw e;
            }
        } else {
            // 不指定名称，Fun Call查找
            if (log.isDebugEnabled()) {
                log.debug("Mcp resources read runtime={}-{}", uri, mcpRuntime);
            }
            Assert.notNull(mcpRuntime, "Mcp resources read runtime can not be empty: " + client);
            // 构建Fun Call
            List<ProviderFunCall> funCalls = new ArrayList<ProviderFunCall>();
            funCalls.addAll(this.resourcesTemplatesList(client, mcpDimension));
            funCalls.addAll(this.resourcesList(client, mcpDimension));
            SyncConfig syncConfig = SyncConfig.builder()
                    // 构建特殊查询
                    .metadata(ImmutableMap.of(ProviderRequestService.KEY_FUN_INTERNAL, funCalls))
                    .workflow(ProviderRequestService.KEY_FUN_SELECT)
                    .reQuery(mcpRuntime.getWorkTask().getQuery())
                    .timeout(mcpRuntime.getTimeout(this.timeout))
                    .workTask(mcpRuntime.getWorkTask())
                    .build();
            // Fun Call模型选择
            String response = SyncWorkflowTask.exeWorkflow(this.notifierService, syncConfig).get();
            if (log.isDebugEnabled()) {
                log.debug("Mcp resources read (fun call) original response={}-{}", uri, response);
            }
            McpResult<String> result = new McpResult<String>();
            result.setResult(response);
            result.setClient(client);
            result.setName(uri);
            if (log.isDebugEnabled()) {
                log.debug("Mcp resources read {} took {} milliseconds to get fun call result", client, watcher.getConsuming());
            }
            if (this.eventListener != null) {
                // 内部事件监听并返回
                this.eventListener.listen(new McpEvent(mcpDimension, result, client, uri));
            }
            return result;
        }
    }

    public McpResult<String> resourcesRead(String client, String uri, McpDimension mcpDimension) throws Exception {
        Assert.hasText(client, "Mcp client can not be empty");
        Assert.hasText(uri, "Mcp uri can not be empty");
        String[] pair = this.namesService.decode(client);
        Assert.isTrue(pair.length == 2, "Mcp resource name is invalid");
        return this.resourcesRead(pair[0], uri, null, mcpDimension);
    }

    // 将PromptGet结果构建为记忆
    protected List<History> buildHistories(List<Map<String, Object>> messages) {
        List<History> histories = new ArrayList<History>();
        for (Map<String, Object> message : messages) {
            try {
                Map<String, Object> content = Map.class.cast(message.get("content"));
                String role = String.class.cast(message.get("role"));
                Assert.notEmpty(content, "Mcp prompt content can not be empty");
                Assert.hasText(role, "Mcp prompt role can not be empty");
                String type = String.class.cast(content.get("type"));
                Assert.hasText(type, "Mcp prompt type can not be empty");
                String text = McpContentUtils.resource(type, content);
                Assert.hasText(text, "Mcp prompt text can not be empty");
                History history = new History();
                history.setContent(text);
                if (role.equalsIgnoreCase("user")) {
                    history.setQuery();
                    history.setUser();
                } else {
                    history.setAssistant();
                    history.setAnswer();
                }
                histories.add(history);
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
        return histories;
    }

    public static class McpClientFactory extends BasePooledObjectFactory<McpClient> {

        protected final CloseableHttpAsyncClient other;

        protected final Map<String, String> headers;

        protected final Map<String, String> env;

        protected final String command;

        protected final String[] args;

        protected final String name;

        protected final String type;

        protected final String url;

        public McpClientFactory(CloseableHttpAsyncClient other, String name, Map<String, Object> value) {
            List<String> args = List.class.cast(value.get("args"));
            this.args = !CollectionUtils.isEmpty(args) ? args.toArray(new String[]{}) : null;
            this.url = StringUtils.defaultString(String.class.cast(value.get("baseUrl")), String.class.cast(value.get("url")));
            this.command = String.class.cast(value.get("command"));
            this.headers = Map.class.cast(value.get("headers"));
            this.type = String.class.cast(value.get("type"));
            this.env = Map.class.cast(value.get("env"));
            this.other = other;
            this.name = name;
            if (log.isInfoEnabled()) {
                log.info("Create mcp client: name={}, value={}", name, value);
            }
        }

        @Override
        public McpClient create() throws Exception {
            if (StringUtils.isEmpty(this.type) || McpClientServiceImpl.TYPE_STDIO.equalsIgnoreCase(this.type)) {
                Assert.hasText(this.command, "Command can not be empty");
                if (log.isDebugEnabled()) {
                    log.debug("Create mcp stdio client: name={},command={}", this.name, this.command);
                }
                return new McpStdioClient(this.name, this.env, this.command, this.args);
            }
            if (McpClientServiceImpl.TYPE_STREAM.equalsIgnoreCase(this.type)) {
                Assert.hasText(this.url, "HttpURL can not be empty");
                if (log.isDebugEnabled()) {
                    log.debug("Create mcp http client: name={},url={}", this.name, this.url);
                }
                return new McpStreamClient(this.other, this.name, this.url, this.headers);
            }
            throw new WorkflowException("MCP client creation failed: " + this.type);
        }

        @Override
        public PooledObject<McpClient> wrap(McpClient mcpClient) {
            return new DefaultPooledObject<McpClient>(mcpClient);
        }

        @Override
        public void destroyObject(PooledObject<McpClient> p) {
            IOUtils.closeQuietly(p.getObject());
            if (log.isInfoEnabled()) {
                log.info("The mcp client destroyed, name={}, url={}", this.name, this.url);
            }
        }
    }

    @Getter
    @Builder
    public static class McpConfig {

        protected GenericObjectPool<McpClient> client;

        protected Integer timeout;
    }

    public static class McpFuture implements Callable<Object> {

        protected final GenericObjectPool<McpClient> client;

        protected final Integer borrow;

        protected final String method;

        protected final Object[] args;

        public McpFuture(GenericObjectPool<McpClient> client, Integer borrow, String method, Object[] args) {
            this.client = client;
            this.borrow = borrow;
            this.method = method;
            this.args = args;
        }

        @Override
        public Object call() throws Exception {
            McpClient mcpClient = null;
            try {
                mcpClient = this.client.borrowObject(this.borrow);
                Object result = MethodUtils.invokeMethod(mcpClient, this.method, this.args);
                this.client.returnObject(mcpClient);
                return result;
            } catch (Exception e) {
                if (mcpClient != null) {
                    this.client.invalidateObject(mcpClient);
                }
                throw e;
            }
        }
    }

    @ConditionalOnProperty(name = "mcp.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected PlaceholderResolver placeholderResolver;

        @Autowired
        @Qualifier("tools")
        protected CloseableHttpAsyncClient httpClient;

        @Autowired(required = false)
        protected EventListenerService eventListener;

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Autowired
        protected NotifierService notifierService;

        @Autowired
        protected NamesService namesService;

        @Value("${mcp.timeBetweenEvictionRunsMillis:30000}")
        protected Integer timeBetweenEvictionRunsMillis;

        @Value("${mcp.processor:}")
        // MCP的线程池数量
        protected Integer processor;

        @Value("${mcp.timeout:1800000}")
        // MCP的执行超时
        protected Integer timeout;

        @Value("${mcp.borrow:60000}")
        // MCP的获取超时
        protected Integer borrow;

        @Bean
        @ConditionalOnMissingBean(value = McpClientService.class)
        public McpClientService mcpClientService() throws Exception {
            // 初始化处理数量
            this.processor = this.processor != null ? this.processor : Runtime.getRuntime().availableProcessors();
            McpClientServiceImpl mcpClientService = new McpClientServiceImpl();
            BeanUtils.copyProperties(this, mcpClientService);
            log.info("McpClientServiceImpl inited, processor={}, timeout={} ,borrow={}", mcpClientService.getProcessor(), mcpClientService.getTimeout(), mcpClientService.getBorrow());
            return mcpClientService;
        }
    }
}
