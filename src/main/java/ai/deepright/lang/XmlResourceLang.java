package ai.deepright.lang;

import static org.springframework.util.StringUtils.hasText;


import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;

import java.util.Locale;
import java.util.ResourceBundle;

@Slf4j
public final class XmlResourceLang {

    public static final XmlResourceControl XML_CONTROL = new XmlResourceControl();

    public static final String DEFAULT_BASE_NAME = "Lang.messages";

    private XmlResourceLang() {
    }

    public static String get(String key) {
        return XmlResourceLang.get(key, Locale.getDefault());
    }

    public static String get(Class<?> clazz, String key) {
        return XmlResourceLang.get(clazz.getName() + "." + key, Locale.getDefault());
    }

    public static String get(Class<?> clazz, String key, Locale locale) {
        return XmlResourceLang.get(XmlResourceLang.DEFAULT_BASE_NAME, clazz.getName() + "." + key, locale, XmlResourceLang.buildClassLoader());
    }

    public static String get(String key, Locale locale) {
        return XmlResourceLang.get(XmlResourceLang.DEFAULT_BASE_NAME, key, locale, XmlResourceLang.buildClassLoader());
    }

    public static String get(Class<?> clazz, String baseName, String key, Locale locale) {
        return XmlResourceLang.get(baseName, clazz.getName() + "." + key, locale, XmlResourceLang.buildClassLoader());
    }

    public static String get(String baseName, String key, Locale locale) {
        return XmlResourceLang.get(baseName, key, locale, XmlResourceLang.buildClassLoader());
    }

    public static String get(Class<?> clazz, String baseName, String key, Locale locale, ClassLoader loader) {
        return XmlResourceLang.get(baseName, clazz.getName() + "." + key, locale, loader);
    }

    public static String get(String baseName, String key, Locale locale, ClassLoader loader) {
        try {
            WorkflowException.check(!hasText(baseName), "The baseName could not find resource: " + baseName, ProtocolCode.C400);
            WorkflowException.check(locale == null, "The locale could not find resource: " + locale, ProtocolCode.C400);
            WorkflowException.check(loader == null, "The loader could not find resource: " + loader, ProtocolCode.C400);
            WorkflowException.check(!hasText(key), "The key could not find resource: " + key, ProtocolCode.C400);
            ResourceBundle bundle = ResourceBundle.getBundle(baseName, locale, loader, XmlResourceLang.XML_CONTROL);
            return StringUtils.defaultIfEmpty(bundle.getString(key), "");
        } catch (Exception e) {
            log.error(e.getMessage(), e);
            return "The key is unresolved: " + key;
        }
    }

    public static void clearCache() {
        ResourceBundle.clearCache(XmlResourceLang.buildClassLoader());
    }

    public static ClassLoader buildClassLoader() {
        ClassLoader cl = XmlResourceLang.class.getClassLoader();
        if (cl != null) {
            return cl;
        }
        cl = Thread.currentThread().getContextClassLoader();
        return cl != null ? cl : ClassLoader.getSystemClassLoader();
    }
}
