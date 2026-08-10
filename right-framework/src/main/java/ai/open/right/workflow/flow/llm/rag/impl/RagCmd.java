package ai.open.right.workflow.flow.llm.rag.impl;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.command.QuickCommand;
import ai.open.right.workflow.flow.command.QuickCommandStore;
import ai.open.right.workflow.flow.llm.rag.RagCondition;
import ai.open.right.workflow.flow.llm.rag.RagConfig;
import ai.open.right.workflow.flow.llm.rag.RagData;
import ai.open.right.workflow.flow.llm.rag.RagService;
import ai.open.right.workflow.flow.llm.rag.future.RagAtOnce;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlElementWrapper;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlProperty;
import com.fasterxml.jackson.dataformat.xml.annotation.JacksonXmlRootElement;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;
import org.springframework.util.CollectionUtils;
import org.springframework.util.StringUtils;

import java.util.Comparator;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Slf4j
@Setter
@Getter
// 使用快捷指令增强内容
public class RagCmd extends RagCondition implements RagService {

    public static final String RAG_KEY = "rag_cmd";

    protected QuickCommandStore quickCommandStore;

    @Override
    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception {
        if (!this.allowed(ragConfig, ragData)) {
            return RagFuture.NOTHING;
        }
        if (log.isDebugEnabled()) {
            log.debug("Rag cmd start");
        }
        try {
            Assert.notNull(this.quickCommandStore, "The command store can not be empty, please config `command.enable`");
            List<QuickCommand> commands = this.quickCommandStore.restore(ragData.getQuery().getBiz(), ragData.getQuery().getChat(), ragData.getQuery().getUserContext().getDevice());
            if (!CollectionUtils.isEmpty(commands)) {
                if (ragConfig.isOverride()) {
                    // Override，是否替换Query
                    String query = this.findPriorityCmd(ragData.getQuery().getQuery(), commands);
                    if (log.isInfoEnabled()) {
                        log.info("Replace query={}", query);
                    }
                    if (StringUtils.hasText(query)) {
                        ragData.getQuery().setQuery(query);
                    }
                } else {
                    RagService.updatePrompt(ragConfig, ragData, ragConfig.getReplace(), this.buildCmd(ragConfig, commands));
                }
            }
        } catch (Exception e) {
            WorkflowException.dolog(e);
        }
        return new RagAtOnce(ragConfig);
    }

    // 构建Cmd
    protected Object buildCmd(RagConfig ragConfig, List<QuickCommand> quickCommands) throws Exception {
        if (ragConfig.isMode(RagConfig.MODE_JSON) || CollectionUtils.isEmpty(quickCommands)) {
            return quickCommands;
        }
        return new LLMCommandsPrompts(quickCommands);
    }

    // 获取优先级最高的匹配Cmd（equalsIgnoreCase）
    protected String findPriorityCmd(String query, List<QuickCommand> commands) throws Exception {
        return commands.stream().filter(command -> query.equalsIgnoreCase(command.getCommand()))
                .max(Comparator.comparingLong(QuickCommand::getPriority))
                .map(QuickCommand::getContent).orElse(query);
    }


    @Getter
    @JacksonXmlRootElement(localName = "Commands")
    public static class LLMCommandsPrompts {

        @JacksonXmlElementWrapper(useWrapping = false)
        @JacksonXmlProperty(localName = "Command")
        protected Map<String, String> command;

        public LLMCommandsPrompts(List<QuickCommand> quickCommands) {
            quickCommands.sort((o1, o2) -> Long.compare(o2.getPriority(), o1.getPriority()));
            for (QuickCommand quickCommand : quickCommands) {
                this.add(quickCommand);
            }
        }

        public LLMCommandsPrompts add(QuickCommand quickCommand) {
            if (this.command == null) {
                this.command = new HashMap<String, String>();
            }
            this.command.putIfAbsent(quickCommand.getCommand(), quickCommand.getContent());
            return this;
        }
    }

    @ConditionalOnProperty(name = "command.enable", havingValue = "true", matchIfMissing = false)
    @Configuration
    @Setter
    @Getter
    public static class InitConfig extends ConditionInitConfig {

        @Autowired(required = false)
        protected QuickCommandStore quickCommandStore;

        @Bean(RagCmd.RAG_KEY)
        @ConditionalOnMissingBean(name = RagCmd.RAG_KEY)
        public RagCmd ragCmd() throws Exception {
            RagCmd ragCmd = new RagCmd();
            BeanUtils.copyProperties(this, ragCmd);
            log.info("RagCmd inited. timeout4Condition={}", ragCmd.getTimeout4Condition());
            return ragCmd;
        }
    }
}
