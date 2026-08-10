package ai.open.right.workflow.flow.config;

import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.util.CollectionUtils;

import java.util.ArrayList;
import java.util.List;

@Slf4j
@Getter
@Setter
public class AllowedConfig extends GlobalConfig {

    // 配置白名单后黑名单无效
    protected List<String> whiteList;

    // 配置白名单后黑名单无效
    protected List<String> blackList;

    public AllowedConfig addWhite(String white) {
        this.whiteList = this.whiteList != null ? this.whiteList : new ArrayList<String>();
        this.whiteList.add(white);
        return this;
    }

    public AllowedConfig addBlack(String black) {
        this.blackList = this.blackList != null ? this.blackList : new ArrayList<String>();
        this.blackList.add(black);
        return this;
    }

    public Boolean allowed(String name) {
        // 过滤名称检查
        if (!CollectionUtils.isEmpty(this.whiteList)) {
            for (String each : this.whiteList) {
                if (name.matches(each)) {
                    if (log.isDebugEnabled()) {
                        log.debug("Fun Call has been added to the whitelist, please check if the method is in the whitelist. for example, BIZ@WORKFLOW");
                    }
                    // 配置了白名单，在白名单
                    return true;
                }
            }
            // 配置了白名单，不在白名单
            return false;
        }
        if (!CollectionUtils.isEmpty(this.blackList)) {
            for (String each : this.blackList) {
                if (name.matches(each)) {
                    if (log.isDebugEnabled()) {
                        log.debug("Fun Call has been added to the blacklist, please check if the method is in the blacklist. for example, BIZ@WORKFLOW");
                    }
                    // 配置了黑名单，在黑名单
                    return false;
                }
            }
            // 配置了黑名单，不在黑名单
            return true;
        }
        // 没配置任何名单
        return true;
    }
}
