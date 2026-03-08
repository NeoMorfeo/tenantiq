import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { z } from "zod";
import { zodErrorMap } from "./zod-error-map";
import en from "./locales/en/translation.json";
import es from "./locales/es/translation.json";
import zodEn from "./locales/en/zod.json";
import zodEs from "./locales/es/zod.json";

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en, zod: zodEn },
      es: { translation: es, zod: zodEs },
    },
    fallbackLng: "en",
    interpolation: { escapeValue: false },
    detection: {
      order: ["localStorage", "navigator"],
      caches: ["localStorage"],
      lookupLocalStorage: "tenantiq_lang",
    },
  });

// Use i18next-based error map for Zod validation messages.
z.setErrorMap(zodErrorMap);

export default i18n;
