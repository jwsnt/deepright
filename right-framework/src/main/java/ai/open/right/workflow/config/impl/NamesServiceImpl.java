package ai.open.right.workflow.config.impl;

import ai.open.right.WorkflowException;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.config.NamesService;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.codec.digest.DigestUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

@Slf4j
@Setter
@Getter
// 用于Fun Call方法重命名
public class NamesServiceImpl implements NamesService {

    //https://cloud.google.com/vertex-ai/generative-ai/docs/multimodal/function-calling?hl=zh-cn#best-practices
    // 编码可用字符集
    protected static final String ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";

    protected static final int MASK = NamesServiceImpl.ALPHABET.length() - 1;

    protected final Map<String, String> nameMapping = new ConcurrentHashMap<String, String>();

    // 编码缓存
    protected final Map<String, String> nameCache = new HashMap<String, String>();

    public final static String SPLIT = "__";

    // 是否开启编码
    protected Boolean encode = true;

    // Fun Call编码长度
    protected Integer length = 32;

    protected String workflow;

    protected String resource;

    protected String prompt;

    protected String tools;

    @PostConstruct
    public void init() throws Exception {
        this.nameMapping.put(NamesServiceImpl.PREFIX_RESOURCE, StringUtils.defaultIfEmpty(this.resource, NamesServiceImpl.PREFIX_RESOURCE));
        this.nameMapping.put(NamesServiceImpl.PREFIX_WORKFLOW, StringUtils.defaultIfEmpty(this.workflow, NamesServiceImpl.PREFIX_WORKFLOW));
        this.nameMapping.put(NamesServiceImpl.PREFIX_PROMPT, StringUtils.defaultIfEmpty(this.prompt, NamesServiceImpl.PREFIX_PROMPT));
        this.nameMapping.put(NamesServiceImpl.PREFIX_TOOLS, StringUtils.defaultIfEmpty(this.tools, NamesServiceImpl.PREFIX_TOOLS));
    }

    // 内部编码方法
    protected String encode(String input) throws Exception {
        if (!this.encode || this.isPrefix(input)) {
            // 不需要编码或者已经编码
            if (log.isDebugEnabled()) {
                log.debug("Encoding is not needed");
            }
            return input;
        }
        StringBuffer result = new StringBuffer();
        int value = 0;
        int bits = -8;
        for (byte b : DigestUtils.sha256(input)) {
            value = (value << 8) | (b & 0xFF);
            bits += 8;
            while (bits >= 6) {
                result.append(NamesServiceImpl.ALPHABET.charAt((value >>> bits) & NamesServiceImpl.MASK));
                bits -= 6;
            }
        }
        if (bits > 0) {
            result.append(NamesServiceImpl.ALPHABET.charAt(((value << (6 - bits)) & 0x3F)));
        }
        return result.substring(0, this.length);
    }

    // 使用指定前缀，客户端，方法名编码
    public String encode(String prefix, String client, String name) throws Exception {
        Assert.isTrue(!StringUtils.contains(client, NamesServiceImpl.SPLIT), "Tools client disallows the '__': " + client);
        Assert.isTrue(!StringUtils.contains(name, NamesServiceImpl.SPLIT), "Tools name disallows the '__': " + name);
        String val = client + NamesServiceImpl.SPLIT + name;
        String key = this.encode(val);
        if (!this.nameCache.containsKey(key)) {
            // 不使用Val
            this.nameCache.putIfAbsent(key, val);
        }
        return this.nameMapping.get(prefix) + key;
    }

    // 使用指定前缀，编码名解码为客户端+方法
    protected String[] decode(String prefix, String name) throws Exception {
        // 去除前缀（精确匹配）
        String key = name.replace(prefix, "");
        String val = this.nameCache.get(key);
        if (StringUtils.isEmpty(val)) {
            throw new WorkflowException("The specified tool is non-existent; please verify the context: " + key, ProtocolCode.C915).needSilent();
        }
        return val.split(NamesServiceImpl.SPLIT);
    }

    // 编码名解码为客户端+方法
    public String[] decode(String name) throws Exception {
        String resource = this.nameMapping.get(NamesServiceImpl.PREFIX_RESOURCE);
        if (StringUtils.startsWithIgnoreCase(name, resource)) {
            return this.decode(resource, name);
        }
        String workflow = this.nameMapping.get(NamesServiceImpl.PREFIX_WORKFLOW);
        if (StringUtils.startsWithIgnoreCase(name, workflow)) {
            return this.decode(workflow, name);
        }
        String prompt = this.nameMapping.get(NamesServiceImpl.PREFIX_PROMPT);
        if (StringUtils.startsWithIgnoreCase(name, prompt)) {
            return this.decode(prompt, name);
        }
        // 剩余均使用TOOLS（兼容偶尔的解析错误）
        return this.decode(this.nameMapping.get(NamesServiceImpl.PREFIX_TOOLS), name);
    }

    public Boolean isPrefixWorkflow(String name) throws Exception {
        return name.startsWith(this.nameMapping.get(NamesServiceImpl.PREFIX_WORKFLOW));
    }

    public Boolean isPrefixResource(String name) throws Exception {
        return name.startsWith(this.nameMapping.get(NamesServiceImpl.PREFIX_RESOURCE));
    }

    public Boolean isPrefixPrompt(String name) throws Exception {
        return name.startsWith(this.nameMapping.get(NamesServiceImpl.PREFIX_PROMPT));
    }

    public Boolean isPrefixTools(String name) throws Exception {
        return name.startsWith(this.nameMapping.get(NamesServiceImpl.PREFIX_TOOLS));
    }

    public Boolean isPrefix(String name) throws Exception {
        return this.isPrefixTools(name) || this.isPrefixPrompt(name) || this.isPrefixResource(name) || this.isPrefixWorkflow(name);
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Value("${names.encode:false}")
        // 是否开启编码
        protected Boolean encode = false;

        // Fun Call编码长度
        @Value("${names.length:8}")
        protected Integer length = 8;

        @Value("${names.workflow:}")
        protected String workflow;

        @Value("${names.resource:}")
        protected String resource;

        @Value("${names.prompt:}")
        protected String prompt;

        @Value("${names.tools:}")
        protected String tools;

        @Bean
        @ConditionalOnMissingBean(value = NamesService.class)
        public NamesService namesService() throws Exception {
            NamesServiceImpl namesService = new NamesServiceImpl();
            BeanUtils.copyProperties(this, namesService);
            log.info("NamesService inited: length={},encode={}", namesService.getLength(), namesService.getEncode());
            return namesService;
        }
    }
}
