package ai.deepright.utils;

import java.util.Arrays;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

public class TemplateChecker {

    public static final Pattern PLACEHOLDER_PATTERN = Pattern.compile("\\$\\{[^}]+}");

    public static final String[] PLACEHOLDER = Arrays.asList(
            "#skill_usage",
            "#browser",
            "#remote",
            "#image",
            "#email",
            "#feishu",
            "#deepright",
            "#creator",
            "#agentId",
            "#answer",
            "#app",
            "#artifact",
            "#base",
            "#chat",
            "#content",
            "#device",
            "#plugin",
            "#git",
            "#identity",
            "#index",
            "#knowledge",
            "#lastUpdate",
            "#main",
            "#memory",
            "#plugin",
            "#path",
            "#plan",
            "#provider",
            "#query",
            "#read",
            "#recall",
            "#router",
            "#rules",
            "#safety",
            "#schema",
            "#skill_create",
            "#skill_extract",
            "#skills",
            "#soul",
            "#source",
            "#sys",
            "#target",
            "#team",
            "#terminal",
            "#timezone",
            "#tools",
            "#tools_cli",
            "#tools_skill",
            "#user",
            "#write",
            "#why",
            "#workspace"
    ).toArray(new String[0]);

    public static final Pattern HASH_PATTERN = Pattern.compile(
            "(?i)(?<![A-Za-z0-9_])(?:"
                    + Arrays.stream(TemplateChecker.PLACEHOLDER)
                    .map(Pattern::quote)
                    .collect(Collectors.joining("|"))
                    + ")(?![A-Za-z0-9_])"
    );

    // True为未发现（没问题）
    public static Boolean check(String template) {
        return !TemplateChecker.HASH_PATTERN.matcher(template).find() && !TemplateChecker.PLACEHOLDER_PATTERN.matcher(template).find();
    }
}
