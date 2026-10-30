import type { ReactNode } from 'react'
import { site, whatsappLink } from '../lib/site'

const UPDATED = '30.10.2026'

export function PrivacyPage() {
  const operator = site.operatorName
    ? `${site.operatorName}${site.operatorBin ? `, ИИН/БИН ${site.operatorBin}` : ''}`
    : `владелец магазина ${site.name}`

  return (
    <article className="container-page max-w-3xl pt-8">
      <h1 className="text-4xl sm:text-5xl">Политика конфиденциальности</h1>
      <p className="mt-3 text-sm text-muted">Редакция от {UPDATED}</p>

      <div className="mt-8 space-y-8 leading-relaxed">
        <Section title="Кто обрабатывает данные">
          <p>
            Оператор персональных данных: {operator}, г. {site.city}, {site.operatorAddress}. Мы
            обрабатываем данные в соответствии с Законом Республики Казахстан «О персональных данных
            и их защите».
          </p>
        </Section>

        <Section title="Какие данные мы получаем">
          <p>
            Когда вы оформляете заказ: имя, номер телефона, способ доставки, адрес (если нужна
            доставка) и комментарий к заказу. Мы не собираем данные банковских карт: оплата проходит
            через Kaspi.
          </p>
        </Section>

        <Section title="Зачем">
          <ul className="list-disc space-y-1 pl-5">
            <li>чтобы подтвердить и выполнить заказ, выставить счёт и доставить покупку;</li>
            <li>чтобы связаться с вами по заказу в WhatsApp или по телефону;</li>
            <li>чтобы видеть историю ваших заказов и помогать с выбором ухода;</li>
            <li>
              только если вы отметили отдельное согласие: чтобы сообщать об акциях и новинках в
              WhatsApp.
            </li>
          </ul>
        </Section>

        <Section title="Где и сколько хранятся">
          <p>
            Данные хранятся на сервере в Республике Казахстан. Мы храним их, пока вы пользуетесь
            магазином, и удаляем по вашему запросу, кроме сведений, которые закон обязывает хранить
            для учёта продаж.
          </p>
        </Section>

        <Section title="Кому передаются">
          <p>
            Мы не продаём и не передаём ваши данные третьим лицам для рекламы. Адрес и телефон
            получает только служба доставки, когда это нужно для отправки вашего заказа.
          </p>
        </Section>

        <Section title="Ваши права">
          <p>
            Вы можете узнать, какие данные о вас хранятся, исправить их, отказаться от рассылки или
            попросить удалить данные. Напишите нам в{' '}
            <a href={whatsappLink()} target="_blank" rel="noreferrer" className="underline">
              WhatsApp {site.whatsappDisplay}
            </a>
            , мы ответим в течение нескольких дней.
          </p>
        </Section>
      </div>
    </article>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h2 className="mb-3 text-2xl">{title}</h2>
      {children}
    </section>
  )
}
