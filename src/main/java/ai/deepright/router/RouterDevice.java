package ai.deepright.router;

import ai.deepright.feature.FeatureField;
import ai.deepright.feature.FeatureUtils;
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

import java.io.File;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

@Getter
@Setter
@Slf4j
@ToString
public class RouterDevice extends RouterSchema {

    // 团队联系人，用以召回外部团队
    public static final String KEY_ROUTER_REMOTE = "router_remote";

    protected Long created = System.currentTimeMillis();

    protected Map<String, Object> metadata;

    // 由RouterAgent提供
    protected Boolean enabled;

    // 由RouterAgent提供
    protected String provider;

    public RouterDevice(WorkflowTask workTask, String device, String agent) throws Exception {
        // 没有传递Name或为空，则使用当前WorkTask
        this.metadata = workTask.getMetadata();
        this.sys = FeatureUtils.buildSys(workTask);
        this.gateway = FeatureUtils.buildGateway(workTask);
        this.terminal = FeatureUtils.buildTerminal(workTask);
        this.workspace = FeatureUtils.buildWorkspace(workTask);
        this.device = StringUtils.defaultIfEmpty(device, workTask.getDevice());
        this.agent = StringUtils.defaultIfEmpty(agent, RouterDevice.agent(workTask));
        if ((StringUtils.isEmpty(this.agent) || StringUtils.isEmpty(this.description)) && log.isDebugEnabled()) {
            log.debug("The router name={} or desc={} is empty", this.agent, this.description);
        }
    }

    public RouterDevice(WorkflowTask workTask, String agent) throws Exception {
        this(workTask, workTask.getDevice(), agent);
    }

    public RouterDevice(WorkflowTask workTask) throws Exception {
        this(workTask, workTask.getDevice(), RouterDevice.agent(workTask));
    }

    public RouterDevice() throws Exception {

    }

    // 内网
    public Boolean isInternal(RouterDevice routerDevice) throws Exception {
        return StringUtils.equalsIgnoreCase(this.gateway, routerDevice.getGateway());
    }

    // 单机
    public Boolean isLoop(RouterDevice routerDevice) throws Exception {
        return StringUtils.equalsIgnoreCase(this.device, routerDevice.getDevice());
    }

    public Boolean isLoop(WorkflowTask workTask) throws Exception {
        return StringUtils.equalsIgnoreCase(this.device, workTask.getDevice());
    }

    public Boolean isSame(RouterDevice routerDevice) throws Exception {
        return StringUtils.equals(this.device, routerDevice.getDevice()) && StringUtils.equals(this.getAgent(), routerDevice.getAgent());
    }

    public Boolean isSame(WorkflowTask workTask) throws Exception {
        return StringUtils.equals(this.device, workTask.getDevice()) && StringUtils.equals(this.getAgent(), RouterDevice.agent(workTask));
    }

    public Boolean isExpired(Integer expired) throws Exception {
        return (System.currentTimeMillis() - expired) > this.created;
    }

    public String key() throws Exception {
        return StringUtils.join(new String[]{this.device, this.agent}, SplitUtils.SPLIT_AT);
    }

    public RouterDevice resetWorkspace() throws Exception {
        this.workspace = MapUtils.getString(this.metadata, FeatureField.KEY_WORKSPACE);
        return this;
    }

    public RouterDevice maskWorkspace() throws Exception {
        this.workspace = this.device + File.separator + this.agent;
        return this;
    }

    public RouterDevice printRouter() throws Exception {
        if (log.isInfoEnabled()) {
            log.info("The router device={}, agent={}", this.device, this.agent);
        }
        return this;
    }

    public static List<String> contact(WorkflowTask workTask, List<String> defContact) throws Exception {
        List<String> remote = List.class.cast(MapUtils.getObject(workTask.getMetadata(), RouterDevice.KEY_ROUTER_REMOTE));
        remote = remote != null ? new ArrayList<String>(remote) : new ArrayList<String>();
        // 加入自己和默认联系人
        remote.add(workTask.getDevice());
        if (!CollectionUtils.isEmpty(defContact)) {
            remote.addAll(defContact);
        }
        return remote.stream()
                .filter(StringUtils::isNotEmpty)
                .distinct()
                .collect(Collectors.toList());
    }

    public static Boolean disable(WorkflowTask workTask) throws Exception {
        return MapUtils.getBoolean(workTask.getMetadata(), FeatureField.KEY_ROUTER_DISABLE, false);
    }

    public static String agent(WorkflowTask workTask) throws Exception {
        return StringUtils.defaultIfEmpty(MapUtils.getString(workTask.getMetadata(), FeatureField.KEY_AGENTID), "");
    }

    public static String key(WorkflowTask workTask) throws Exception {
        return StringUtils.join(new String[]{workTask.getDevice(), RouterDevice.agent(workTask)}, SplitUtils.SPLIT_AT);
    }

    // 基础Markdown
    public static String buildMarkdown(List<RouterDevice> routerDevices) throws Exception {
        StringBuffer buffer = new StringBuffer();
        // Device/Agent/Desc/Gateway/Workspace/
        buffer.append("|The member device unique ID");
        buffer.append("|The member agent unique ID");
        buffer.append("|The member description, including purpose and functions|");
        buffer.append("|The same gateway MAC address (e.g., 9c:a6:15:5:3f:ba) indicates that the devices are on the same network");
        buffer.append("|The working directory of the device");
        buffer.append("|The terminal execution environment of the device (e.g., zs)");
        buffer.append("|The operating system of the device (e.g., Darwin 23.4.0)");
        buffer.append("|").append(System.lineSeparator());
        buffer.append("|---|---|---|---|---|---|---|");
        buffer.append(System.lineSeparator());
        StringBuffer content = new StringBuffer();
        for (RouterDevice routerDevice : routerDevices) {
            content.append("|").append(routerDevice.getDevice());
            content.append("|").append(routerDevice.getAgent());
            content.append("|").append(StringUtils.defaultIfEmpty(routerDevice.getDescription(), ""));
            content.append("|").append(StringUtils.defaultIfEmpty(routerDevice.getGateway(), ""));
            content.append("|").append(StringUtils.defaultIfEmpty(routerDevice.getWorkspace(), ""));
            content.append("|").append(StringUtils.defaultIfEmpty(routerDevice.getTerminal(), ""));
            content.append("|").append(StringUtils.defaultIfEmpty(routerDevice.getSys(), ""));
            content.append("|").append(System.lineSeparator());
        }
        buffer.append(content).append(System.lineSeparator());
        return buffer.toString();
    }

}
