package ai.deepright.cli;

import ai.deepright.cli.insert.CliInsert;
import ai.open.right.protocol.ProtocolCode;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.file.FileStore;
import ai.open.right.workflow.flow.file.impl.SysStore;
import ai.open.right.workflow.flow.media.MediaTransferUtils;
import lombok.*;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.StringUtils;
import org.apache.http.HttpResponse;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.client.methods.HttpGet;
import org.apache.http.impl.nio.client.CloseableHttpAsyncClient;
import org.springframework.util.Assert;

import java.io.BufferedInputStream;
import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.List;
import java.util.concurrent.TimeUnit;

@Slf4j
@Getter
@Setter
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class CliPubData {

    public static final String LANG_KEY_PUB_ECHO = "pub.echo";

    public static final Integer SUCCESS = 0;

    public static final Integer FAILED = 1;

    // 纯文本（非二进制）
    public static final String TEXT = "text";

    // URL资源
    public static final String URL = "url";

    protected Integer status;

    // 资源后缀
    protected String suffix;

    @Builder.Default
    // 默认类型TEXT
    protected String encode = CliPubData.TEXT;

    // 附带的补充插入
    protected List<CliInsert> insert;

    // 回传的CliSubOps
    protected String type;

    protected String cmd;

    protected String tid;

    // 强制转换为文本，encode=true开启Base64编码
    public CliPubData forceText(CloseableHttpAsyncClient resource, SysStore sysStore, Integer timeout, Boolean encode) throws Exception {
        if (StringUtils.equalsIgnoreCase(this.encode, CliPubData.URL)) {
            // 检查是否为网络资源
            if (MediaTransferUtils.isNetwork(this.getCmd())) {
                HttpGet httpGet = new HttpGet(this.getCmd());
                try {
                    httpGet.setConfig(RequestConfig.custom()
                            .setConnectionRequestTimeout(timeout)
                            .setSocketTimeout(timeout)
                            .build());
                    HttpResponse response = resource.execute(httpGet, null, null).get(timeout, TimeUnit.MILLISECONDS);
                    // 错误模型需要使用
                    Assert.isTrue(ProtocolCode.range2xx(response.getStatusLine().getStatusCode()), this.getCmd());
                    try (BufferedInputStream input = new BufferedInputStream(response.getEntity().getContent())) {
                        this.setCmd(encode ? Base64.getEncoder().encodeToString(input.readAllBytes()) : IOUtils.toString(input, StandardCharsets.UTF_8));
                    }
                } catch (Exception e) {
                    httpGet.abort();
                    throw e;
                }
            } else {
                // 尝试从文件系统读取
                byte[] file = sysStore.restore(this.getCmd());
                this.setCmd(encode ? Base64.getEncoder().encodeToString(file) : new String(file, StandardCharsets.UTF_8));
            }
            this.setEncode(CliPubData.TEXT);
        }
        return this;
    }

    public CliPubData forceText(CloseableHttpAsyncClient resource, SysStore sysStore, Integer timeout) throws Exception {
        return this.forceText(resource, sysStore, timeout, false);
    }

    // 强制转换为URL
    public CliPubData forceURL(WorkflowTask workTask, FileStore defStore) throws Exception {
        if (StringUtils.equalsIgnoreCase(this.encode, CliPubData.TEXT)) {
            this.setCmd(defStore.store(this.getCmd().getBytes(StandardCharsets.UTF_8), this.getSuffix(), workTask));
            this.setEncode(CliPubData.URL);
        }
        return this;
    }

    public CliPubData valid() throws Exception {
        Assert.isTrue(CliPubData.SUCCESS.equals(this.getStatus()), this.getCmd());
        return this;
    }

    public CliPubData check() throws Exception {
        Assert.notNull(this.suffix, "The cli@pub suffix can not be empty");
        Assert.notNull(this.status, "The cli@pub status can not be empty");
        Assert.hasText(this.cmd, "The cli@pub cmd can not be empty");
        Assert.hasText(this.tid, "The cli@pub tid can not be empty");
        return this;
    }

    public Boolean hasInsert() {
        return !CollectionUtils.isEmpty(this.insert);
    }

    public Boolean isEncode(String encode) {
        return StringUtils.equalsIgnoreCase(this.encode, encode);
    }

    public Boolean isType(String type) {
        return StringUtils.equalsIgnoreCase(this.type, type);
    }

    public Boolean isOk() {
        return CliPubData.SUCCESS.equals(this.getStatus());
    }
}
