package ai.open.right.workflow.config;

import ai.open.right.context.UserContext;
import lombok.*;
import org.springframework.util.Assert;

@Setter
@Getter
@Builder
@ToString
@AllArgsConstructor
public class ConfigSearch {

    protected String biz;

    @Builder.Default
    protected String device = UserContext.UNKNOWN;

    @Builder.Default
    protected String language = UserContext.UNKNOWN;

    public ConfigSearch() {

    }

    public static class ConfigSearchChecker {

        public static void check(ConfigSearch search) {
            Assert.hasText(search.getBiz(), "The biz can not be empty");
            Assert.hasText(search.getDevice(), "The device can not be empty");
            Assert.hasText(search.getLanguage(), "The language can not be empty");
        }
    }
}
