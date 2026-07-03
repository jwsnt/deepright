package ai.deepright.llm.optimize;

import ai.open.right.workflow.flow.llm.LLMQuery;
import ai.open.right.workflow.flow.llm.config.LLMConfig;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.store.history.History;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@Slf4j
@Getter
@Setter
public class RequestDiscard {

    public static final String NAME = "request_discard";

    // 如果疑似的User Query超过指定ms则删除，避免干扰新会话
    protected Integer interval;

    public void discard(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery) throws Exception {
        List<History> histories = providerRequest.getMessage().getHistories();
        if (CollectionUtils.isEmpty(histories)) {
            return;
        }
        providerRequest.getMessage().setHistories(this.compare(providerRequest, llmConfig, llmQuery, this.suspect(providerRequest, llmConfig, llmQuery, histories), histories));
    }

    // 从原始会话中删除疑似会话
    protected List<History> compare(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery, List<History> suspicious, List<History> histories) throws Exception {
        for (History history : suspicious) {
            // 如果疑似的User Query超过指定阈值外则删除，避免干扰新会话
            if (llmQuery.getCreated() - history.getCreated() > this.interval) {
                if (histories.remove(history) && log.isInfoEnabled()) {
                    log.info("The request discard the history={}", history.getCreated());
                }
            }
        }
        return histories;
    }

    // 处理疑似残缺的历史记录（例如宕机故障）
    protected List<History> suspect(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery, List<History> histories) throws Exception {
        Map<String, List<History>> group = new HashMap<String, List<History>>();
        // 确保有问有答
        for (History history : histories) {
            // 按Chat + Conversation 组合
            String key = this.buildKey(providerRequest, llmConfig, llmQuery, history);
            // User则Put, 如果是其他FunCall或Assistant则Remove
            // User Assistant/Tools 则清除
            // User User Assistant/Tools 则清除
            // User Assistant/Tools Assistant/Tools 则清除
            // User User Assistant/Tools Assistant/Tools 则清除
            if (history.isRole(History.ROLE_USER)) {
                List<History> users = group.get(key);
                users = users != null ? users : new ArrayList<History>();
                users.add(history);
                group.put(key, users);
            } else {
                group.remove(key);
            }
        }
        List<History> suspicious = new ArrayList<History>();
        for (List<History> users : group.values()) {
            suspicious.addAll(users);
        }
        return suspicious;
    }

    protected String buildKey(ProviderRequest providerRequest, LLMConfig llmConfig, LLMQuery llmQuery, History history) throws Exception {
        return history.getChat() + history.getConversation();
    }

    @Order(Ordered.LOWEST_PRECEDENCE - 1)
    @Configuration
    @Getter
    @Setter
    public static class InitConfig {

        @Value("${optimize.discard.interval:60000}")
        protected Integer interval;

        @Bean(RequestDiscard.NAME)
        @ConditionalOnMissingBean(name = RequestDiscard.NAME)
        public RequestDiscard requestDiscard() throws Exception {
            RequestDiscard requestDiscard = new RequestDiscard();
            BeanUtils.copyProperties(this, requestDiscard);
            log.info("RequestDiscard inited");
            return requestDiscard;
        }
    }
}
