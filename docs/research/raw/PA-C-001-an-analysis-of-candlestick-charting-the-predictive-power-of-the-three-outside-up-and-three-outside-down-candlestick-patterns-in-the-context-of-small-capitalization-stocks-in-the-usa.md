# An Analysis of Candlestick Charting: the Predictive Power of the Three-Outside-Up and Three-Outside-Down Candlestick Patterns in the Context of Small Capitalization Stocks in the USA

- Source URL: https://open.uct.ac.za/server/api/core/bitstreams/e6c53889-2344-4794-8d72-5f41177a3f36/content
- Capture URL: https://r.jina.ai/http://open.uct.ac.za/server/api/core/bitstreams/e6c53889-2344-4794-8d72-5f41177a3f36/content
- Author/Org: University of Cape Town
- Year: 2015
- Captured At UTC: 2026-03-27T03:31:28.029086+00:00
- Theory: price_action
- Stance: counter

---

# An Analysis of Candlestick Charting: the Predictive Power of the Three-Outside-Up and Three-Outside-Down Candlestick Patterns in the Context of Small Capitalization Stocks in the USA

- Source URL: https://r.jina.ai/http://open.uct.ac.za/server/api/core/bitstreams/e6c53889-2344-4794-8d72-5f41177a3f36/content
- Author/Org: University of Cape Town
- Year: 2015
- Captured At UTC: 2026-03-27T03:31:27.874702+00:00
- Theory: price_action
- Stance: counter

---

Title: thesis_com_2015_hutton_simon.pdf

URL Source: http://open.uct.ac.za/server/api/core/bitstreams/e6c53889-2344-4794-8d72-5f41177a3f36/content

Published Time: Fri, 02 Nov 2018 09:31:28 GMT

Number of Pages: 49

Markdown Content:
University of Cape Town An Analysis of Candlestick Charting: the Predictive Power of the 

## Three-Outside-Up and Three-Outside-Down Candlestick Patterns in the Context of Small Capitalization Stocks in the USA. 

## A research report submitted by Simon Hutton Presented to the Graduate School of Business University of Cape Town 

In partial fulfilment of the requirements for the MCom in Development Finance Degree 

## December 2015 Supervisor: Dr Sean Gossel University of Cape Town The copyright of this thesis vests in the author. No quotation from it or information derived from it is to be published without full acknowledgement of the source. The thesis is to be used for private study or non-commercial research purposes only. 

# Published by the University of Cape Town (UCT) in terms of the non-exclusive license granted to UCT by the author. 1

# Abstract 

This paper examines the predictive power of two Japanese Candlestick patterns for a 49-stock sample of small capitalization stocks drawn from the S&P 600 for the period 1 June 2005 to 15 May 2015. Using the normal approximation to the binomial for statistical testing and a dynamic holding period strategy to test the three-outside-up and three-outside-down patterns, this study contradicts earlier works that used dynamic holding period strategies for large capitalization stocks and showed moderate levels of statistically significant predictive power. This study finds no statistically significant evidence of the predictive power of the three-outside-up and three-outside-down patterns for the sample and time period considered. Hence, the findings imply that there is no evidence to challenge the Efficient Market Hypothesis. 2

# Plagiarism Declaration 

1. I know that plagiarism is wrong. Plagiarism is to use another’s work and pretend that it is one’s own. 

2. I have used convention for citation and referencing. Each contribution to, and quotation in this report from the work of other people has been attributed, and has been cited and referenced. 3. This report is my own work. 4. I have not allowed, and will not allow, anyone to copy my work with the intention of passing it off as his or her own work. 

Signed: Simon Hutton 3

# Acknowledgements 

To my Supervisor, Dr. Sean Gossel, I am grateful for your professionalism and steadfast support in developing this thesis. Your assistance has furthered my appreciation for what thorough analysis entails and has heightened my attention to detail. Thank you Jessica, for your infinite support and patience. Thank you to my Father, Mother and Sister for their ongoing encouragement and positivity. All of you have helped me maintain the momentum and work ethic to complete this work. 4

# Table of Contents 

## Abstract ....................................................................................................................... 1 

## Plagiarism Declaration ................................................................................................. 2 

## Acknowledgements ..................................................................................................... 3 

## Table of Contents ........................................................................................................ 4 

## Chapter 1 - Introduction .............................................................................................. 6 

1.1 Background of the Study ................................................................................................................... 6 

1.2 Problem Definition ............................................................................................................................ 9 

1.3 Research Question, Objective and Scope ....................................................................................... 10 

1.3.1 Research Question.................................................................................................................................. 10 1.3.2 Objective................................................................................................................................................. 10 1.3.3 Scope ...................................................................................................................................................... 10 

1.4 Justification of the Study ................................................................................................................. 11 

1.5 Limitations and Assumptions .......................................................................................................... 11 

1.5.1 Limitations .............................................................................................................................................. 11 1.5.2 Assumptions ........................................................................................................................................... 13 

## Chapter 2 - Literature Review .....................................................................................15 

2.1 Theoretical Basis.............................................................................................................................. 15 

2.1.1 The Efficient Market Hypothesis (EMH) ................................................................................................. 15 2.1.2 The Random Walk Hypothesis................................................................................................................ 16 2.1.3 Behavioral Finance ................................................................................................................................. 17 

2.2 Candlestick Analysis ........................................................................................................................ 18 

2.2.1 General Candlestick Studies ................................................................................................................... 18 2.2.2 Candlestick Studies that Consider the TOU and TOD ............................................................................. 21 5

2.3 Conclusion ....................................................................................................................................... 25 

## Chapter 3 - Research Methodology .............................................................................27 

3.1 Research Design .............................................................................................................................. 27 

3.2 Data Collection & Sample................................................................................................................ 27 

> 3.2.1 Data Collection ....................................................................................................................................... 27 3.2.2 Sample .................................................................................................................................................... 27

3.2 Data Analysis Methods .................................................................................................................... 28  

> 3.2.1 Data and Naming Conventions ............................................................................................................... 28 3.2.2 The Normal Approximation to the Binomial .......................................................................................... 29 3.2.3 Modeling the Normal Approximation to the Binomial........................................................................... 31

## Chapter 4 - Research Findings & Analysis ....................................................................35 

4.1 TOU Event Findings and Analysis .................................................................................................... 35 

4.2 TOD Event Findings and Analysis .................................................................................................... 38 

## Chapter 5 - Research Conclusions & Recommendations ..............................................41 

5.1 Research Conclusions ...................................................................................................................... 41 

5.2 Recommendations for Future Research ......................................................................................... 42 

## References ..................................................................................................................43 

## Appendix I ..................................................................................................................47 6

# Chapter 1 - Introduction 

## 1.1 Background of the Study 

"I absolutely believe that price patterns are being repeated. There are recurring patterns that appear over and ov er, but with slight variations. This is because markets are driven by humans and human nature never changes.” 

-Jesse Livermore (Smitten & Livermore, 2001) Technical analysis uses past prices to predict the future price movements of financial assets such as stocks, bonds, currencies and commodities (Chiang, Ke, & Liang, 2012). Technical analysis involves quantitative and graphical methods with a common set of basic principles and is the most common alternative to fundamental analysis (Fock, Klein, & Zwergel, 2005). The methods of technical analysis are grounded on two key tenets: (i) that trends in prices tend to persist and (ii) that market action is repetitive (Caginalp & Balenovich, 2003). These two tenets infer that price movements in financial markets are neither efficient nor random. A common tool used to conduct technical analysis is candlestick analysis. Candlestick analysis originated in the 

1700’s when a wealthy rice farmer, Munehisa Honma, developed candlestick patterns as a means to analyze forward contracts in the Japanese rice markets (Marshall, Young, & Rose, 2006). Honma developed a system in which each candlestick displays information on market sentiment by expressing graphically the high, low, opening and closing prices. The principles of candlestick analysis were introduced to the Western financial markets in the early 1990’s by Nison (1991) and Morris (1992) who provided investors with a new means to analyze and potentially profit from technical analysis. Nison introduced more than 28 reversal and continuation patterns, while Morris introduced over 50 different reversal and continuation patterns. They claim that, when plotted over time, the price patterns formed by candlesticks can be used to help judge the sentiment of market demand and supply and indicate when reversals or continuations in prices are probable. Candlesticks are charted on an X-Y plane with time on the X-axis and price on the Y-axis. The ‘body ’ of the candlestick displays the price difference between the opening and closing price. If the closing price is higher than the opening price as is the case in an ‘up ’ day, then the body is white whereas if the opening price is higher than the closing price, then the body is black (or a darker color), indicating a ‘down ’ day. Above and below the candlestick body are the ’shadows ’, that indicate the high and the low prices of the trading day. 7Figure 1 graphically illustrates the formation of a candlestick and Figure 2 shows an example of a six-month daily interval candlestick chart for the S&P 500 Index exchange traded fund (ETF). 

Figure 1: Candlestick Open-Close, High-Low 

Source: (Fock et al., 2005) 

> Source: (Fock et al., 2005)

Figure 2: Six-Month Daily Interval Candlestick Chart for the S&P 500 index ETF 

> Source: Bigcharts.com

Since being introduced to Western markets, candlestick charting has been used by financial traders and investors as a part of their analytical toolkit. Despite its widespread use in practice, the empirical evidence on the efficacy of candlestick charting as a tool for predictive analysis is mixed. This study investigates the predictive power of two candlestick reversal patterns. Reversal patterns are intended to indicate when there may be a change in the price trend. If prices were going up and a reversal pattern manifests, it would be anticipated that prices would stop going up and reverse their direction, at least temporarily. Similarly, 8if prices were going down and a reversal pattern manifests, then it would be anticipated that prices would stop going down and reverse direction, at least temporarily. The reversal patterns considered in this study are the three-outside-up (TOU) and three-outside-down (TOD) reversal patterns. 1 As the names imply, the three-outside-up (TOU) and three-outside-down (TOD) are three-day patterns. They are both intended to signal a reversal of a trend. If the price of a stock is in a down trend and a TOU manifests, then it is expected the price will reverse its trend course and go up. Conversely, if the price of a stock is in an up trend and a TOD manifests, then it is expected the price will reverse course and decline. A TOU or TOD is only confirmed in the context of a preceding trend. Hence, a TOU must be preceded by a down trend, while a TOD must be preceded by an up trend. As these patterns are identified in the context of a trend, they are referred to as trend context patterns. 2

Morris (1992) defines the following seven parameters for the TOU. The first day of a TOU is part of a down trend (parameter 1) and is a down day (parameter 2). The second day engulfs the first day (parameter 3) and is an up day (parameter 4), meaning it opens lower than the first days close and closes above the first days open. The third day is an up day (parameter 5) where the open must be higher than the open of the second day (parameter 6) and the close must be higher than the close on the second day (parameter 7). The TOD is the inverse of the TOU. The first day of a TOD is part of an up trend and is an up day, meaning its closing price is higher than its opening price. The second day engulfs the first day and is a down day, meaning it opens higher than the first days close, but closes below the first days open. The third day is a down day where the open must be lower than the open of the second day and the close must be lower than the close on the second day. In the literature, the definition of the TOD and TOU is consistent and is usually defined using a series of inequalities that mathematically define the descriptions of the patterns provided above. The meaning of the inequalities used to define the TOU and TOD is consistent amongst the academic works that consider them. 3

Figure 3(a) and 3(b) show the TOU and TOD as they would appear on a candlestick chart, with the three vertical straight lines indicating the preceding trend and the candlesticks indicating the 3-day pattern.      

> 1The TOU and TOD patterns were first introduced to the western markets in 1992 by Greg Morris, in his influential book Candlepower, advanced candlestick pattern recognition and filtering techniques for trading stocks and futures (Morris, 1992). The TOU and TOD patterns have also been examined in the academic literature, but not exhaustively. The most notable studies have been published by Caginalp and Laurent (1998), Horton (2009) and Lu et al. (2015) who make partly conflicting conclusions. Their studies are further explored in the Literature Review.
> 2This marks an important distinction between studies that use a trend as a precursor to pattern recognition and those that do not. This distinction is consistent with how the TOU and TOD were originally defined and will be further discussed in the literature review.
> 3See the Literature for a description of the key studies.

9

Figure 3(a): Three Outside Up (TOU) Figure 3(b): Three Outside Down (TOD) 

> Source: Caginalp and Laurent (1998)

## 1.2 Problem Definition 

The utility of candlestick analysis as an investment tool is an area of ongoing debate. In their seminal work on candlestick reversal patterns Caginalp and Laurent (1998) highlight that, on the discussion of the utility of technical analysis, there is a substantial rift between the opinions of academics and practitioners. According to the Efficient Market Hypothesis (EMH) past price information is of no use for investment analysis and thus technical analysis and candlestick charting have no value (Chiang et al., 2012; Fama, 1965). However, many finance professionals believe that technical analysis is useful, with 60% of commodity traders and 30% - 40% of foreign exchange traders depending on technical trading systems (Billingsley & Chance, 1996; Menkhoff, 1997). Unfortunately for the study of candlestick analysis as a whole, the answer to the question of its efficacy is likely to remain enigmatic. Candlestick analysis combines dozens of patterns and techniques. 4 Consequently, investigating the utility of candlestick analysis as a whole poses numerous challenges that relate to testing techniques and generality (Park & Irwin, 2007). Effort can be made however, to empirically analyze particular patterns and thus this study considers the predictive power of two patterns that were examined in the early seminal work of Caginalp and Laurent (1998), the TOU and TOD 5.  

> 4According to Morris (1992) there are over 50 different candlestick patterns.
> 5Caginalp and Laurent (1998) who were the first to publish a statistically robust analysis of candlestick charting, found that for four different holding periods, the TOD generated profits more than 73% of the time and the TOU generated profits more than 53% of the time, for stocks in the S&P 500 for the period 19 January 1992 to 14 June 1996.

10 

## 1.3 Research Question, Objective and Scope 

1.3.1 Research Question 

This study investigates the following primary research question: 

Do the TOU and TOD have predictive power for small capitalization stocks within the S&P 600 for the period 1 June 2005 to 15 May 2015 at a moderate level of statistical significance? 

In addition, the analysis will also seek to answer the following sub-question: 

Do the findings on the predictive power of the TOU and TOD for small capitalization stocks within the S&P 600 for the period 1 June 2005 to 15 May 2015 support the Efficient Market Hypothesis at a moderate level of statistical significance? 

The decision to follow this line of enquiry is inspired by the current gap in the academic literature that indicates neither small capitalization stocks, nor the time period highlighted above, have been investigated. Moreover, results of testing the TOU and TOD have been published in the academic literature on only seven occasions. The research questions are also inspired by the seminal work of Caginalp and Laurent (1998) who were the first to pursue a scientifically robust study of the TOD and TOU. 

1.3.2 Objective 

The objective of this study is to empirically determine whether the TOU and TOD have predictive power for small capitalization stocks within the S&P 600 for the period 1 June 2005 to 15 May 2015. This objective will be achieved through a hypothesis test where the null hypothesis (H0) is that the TOD and TOU do not have predictive power for small capitalization stocks within the S&P 600 for the period 1 June 2005 to 15 May 2015 at a moderate level of statistical significance, against the alternative hypothesis (H1) that the TOD and TOU do have predictive power at a moderate level of statistical significance over the same period. 

1.3.3 Scope 

The scope of this study is limited to: 1. The time period of 1 June 2005 to 15 May 2015. This date range covers both bull and bear market conditions thus avoiding market directional biases and provides a sufficient number of data points to address the hypothesis. 11 2. The TOU and TOD candlestick patterns. The investigation of the TOU and TOD patterns affords a narrow focus and offers a contribution to the academic community which has investigated the predictive power of the TOU and TOD only seven times in aggregate and only two times for a period of investigation post 2005. 3. Considering the predictive power of a sample of 47 small-capitalization stocks drawn from the S&P 600 index. The focus on small capitalization stocks in the USA is driven by the gap in the literature. The rationale for selecting 47 small-capitalization stocks from the S&P 600 is expanded upon in the Research Methodology section. 

## 1.4 Justification of the Study 

The topic of this study has been selected due to its relevance to both financial theory and finance practitioners. The theory of the EMH, which excludes technical analysis as a viable investment tool, has been a mainstay of academia and technical finance education for more than four decades (The Institute of Chartered Financial Analysts, 2012). However, more than half of the published academic literature dating back to 1966 suggests that technical analysis is valid as a profitable investment tool (Park & Irwin, 2007). Thus the debate regarding the merits of candlestick analysis is still ongoing. 

## 1.5 Limitations and Assumptions 

1.5.1 Limitations 

1. Predictive Power: This study investigates the predictive power of the TOU and TOD, as will be defined by the change in price of a stock following a TOU or TOD. A mathematical definition of predictive power is established and explained in the Research Methodology section. While this study will comment on profitability of the TOU and TOD in the Research Findings and Analysis section, profitability is not integrated into the hypothesis test. This decision has been informed by the variable profitability of a trade, which is a direct function of ones cost of money, cost of commission, the magnitude of bid-ask spreads upon entry and exit of trade execution, as well as regional taxes and or regulations, all of which vary considerably among investors. Thus defining predictive power by way of estimating profitability is subject to numerous variable elements and is prone to dilute the precision of the findings. 12 2. Results: While the results of this study may reinforce or undermine some tenets of the theory of random walk, the goal of this study is limited to investigating the predictive power of the TOU and TOD and considering how that predictive power, or lack thereof, supports the Efficient Market Hypothesis. 

3. Generalization: Candlestick analysis is a subset of technical analysis, and the TOU and TOD are two of many patterns that constitute candlestick analysis. Hence, the findings of this document cannot be used to generalize support for, or negate technical analysis or candlestick analysis. Rather, the findings are limited to providing insight into two components in the larger field of study of candlestick and technical analysis. 4. Sample: It cannot be known if the sample used is truly representative of all the stocks in the S&P 600 or small capitalization stocks in general. As such, there is a limitation in the representativeness of the sample chosen. However, it is known that the sample provides insight into a specifically defined sample of stocks that meet the selection criteria highlighted in the Research Methodology section. The sample may be subject to a degree of selection bias as it may not represent companies across industries in the exact same proportion as the S&P 600. Such a bias would however require that there was a relationship between industries and the efficacy of candlestick analysis. To date there is no such evidence in the literature. 5. Model: This study employs the normal approximation to the binomial model to test the hypothesis. The properties of this model are generally consistent with the realities of the experiment carried out. A possible exception relates to the assumption that trials are independent, which means that the outcome of one trial does not affect the outcome of any other trials (Anderson, Sweeney, & William, 1993). If the EMH and the theory of random walk are assumed to be valid representations of market price action then there is no conflict because they infer future price movements are independent of past prices, which is consistent with the assumption of trial independence. If however, it is assumed prices are not random, then there is a potential conflict with the independence of trials assumption. The relationship between stocks in the sample may also challenge the assumption of independence among trials. If it were case that there were strong correlations among the stocks in the sample, then it would be expected that the TOU and TOD would each manifest on the same dates with the same outcome. Of the 177 TOU patterns that manifested from the sample considered herein, there were 26 instances in which the TOU occurred on the same date for more than one stock. Of those 26 instances, only ten involved more than two stocks and in only one instance did the TOU occur for more than four stocks 13 simultaneously, out of the 47 stocks in the sample. Of the 221 TOD patterns that manifested, there were 33 instances in which the TOD occurred on the same date for more than one stock. Of those 33 instances, only twelve involved more than two stocks and in only two instances did the TOD occur for more than four stocks simultaneously, out of the 47 stocks in the sample. These instances of shared dates may be attributed to stocks in the same sector having similar price tendencies over time. While an investigation of the correlation among the sample stocks is beyond the scope of this thesis, the above results suggest the correlation is limited. Nevertheless, the issue of independence among trials is the most likely to challenge the validity of using the binomial model in the context of this study. 6. Literature Review: The literature review is limited to academic works that discuss the EMH, theory of random walk, behavioral finance and studies that investigate the predictive power of candlestick patterns using statistical testing. In reviewing academic works that investigate the predictive power of candlestick patterns, the observations are limited to noting, comparing and contrasting the markets considered; the dates over which the markets were evaluated; the holding strategy employed; the candlestick patterns evaluated and the results of tests on the predictive power of those patterns. In reviewing the academic studies on candlestick analysis, several issues that, according to Park and Irwin (2007) can challenge technical analysis testing, are omitted from the discussion. Issues of averaging results, where numerous patterns are considered and only the aggregate average results are stated, are not discussed. Issues of data snooping are also omitted. Data snooping can occur when by fluke or by direct intention, a certain strategy or pattern is found to be valid for a particular set of data (Kilgallen, 2012). Data snooping can be problematic because it can result in authors making broad inferences about new patterns based on only one data set, with no out-of-sample testing (Sullivan, Timmermann, & White, 1999). Issues of standardization are also omitted from discussion. A lack of standardization among testing procedures within the academic literature compounds the challenge of longitudinal comparability and can make definitive inferences about the validity of candlestick patterns difficult. While these concerns are present (albeit scarcely) in the literature considered herein, they are omitted from discussion because they do not directly relate to this thesis. 

1.5.2 Assumptions 

1. It is assumed that the reader has an intermediate knowledge of financial markets and financial markets terminologies. 14 2. It should be assumed that ‘prices ’ refers to the prices of publicly traded securities or assets. 3. It should be assumed that ‘assets ’ refers to publicly traded securities that may include stocks, bonds, currencies and commodities. 4. It is assumed that the normal approximation to the binomial model is an appropriate model to test the hypothesis herein. This model was used in the seminal work of Caginalp and Laurent (1998) and is uncontested in the literature that followed. It is assumed that the basic properties of the binomial model hold 6.

5. It is assumed that the sample selected, while subject to limitations noted in section 1.5.1, provides a reasonable representation of small capitalization stocks that meet predefined criteria, as highlighted in the Research Methodology section. 6. It is assumed that the time period investigated, by way of covering a date range that covers both bull and bear market conditions, is not subject to biases that may come about from testing patterns in only one market condition.         

> 6The basic properties of the binomial model can be summarized as follows (Anderson et al., 1993; Keller, 2009): A binomial experiment consists of a fixed number of trials; on each trial there are two possible outcomes, success and failure; the probability of success is p, the probability of failure is 1-p; the trials are independent, which means that the outcome of one trial does not affect the outcome of any other trials; and the normal distribution may be used to approximate binomial probabilities when np ≥5, and when n(1-p)≥5.

15 

# Chapter 2 - Literature Review 

The purpose of this literature review is to examine the literature on candlestick analysis and the predictive power of the TOU and TOD. The literature review consists of two sections. The first introduces technical analysis in the context of the Efficient Market Hypothesis, random walk and behavioral finance; and the second section discusses the literature pertaining to the predictive power of candlestick analysis in general and as it relates to the TOU and TOD. The literature review then ends with a summary conclusion. 

## 2.1 Theoretical Basis 

The purpose of this section is to provide the reader with a high level appreciation for where candlestick analysis sits in the spectrum of academic theory. This section will consider three theoretical constructs: the Efficient Market Hypothesis (EMH), the theory of random walk and behavioral finance. 7

2.1.1 The Efficient Market Hypothesis (EMH) 

The Efficient Market Hypothesis posits that markets embed all available information into the current price of an asset and that all market participants are rational (Anson, Chambers, Black, & Kazemi, 2012; Fama, 1970). It theorizes that there are three levels of market efficiency, depending on the amount of information that is absorbed by the market: 1. Weak Form Market Efficiency 2. Semi Strong Form Market Efficiency 3. Strong Form Market Efficiency Weak Form efficiency is the most pertinent to the topic of technical analysis. It plainly states that all historical information is embedded into the current price and bears no utility for the investor (Fama, 1970). As such, considering past price information is futile as information contained in past prices is fully reflected in current prices (Marshall et al., 2006). The Semi Strong form of market efficiency suggests that all public information is priced into the market, inferring that even a cunning analyst using public information would not be able to outperform the market (studies by Malkiel (1995) support this notion), while the Strong Form of Market Efficiency suggests that all information, both public and private (insider information) is also reflected in current asset prices. While  

> 7While Park and Irwin (2007) highlight numerous behavioral finance theoretical constructs, including noisy rational expectations, chaos theory, central bank intervention, order flow, temporary market inefficiencies, and market microstructure deficiencies, these will not be considered here. This course of action is guided by the current body of academic work on candlestick analysis that has, to date, omitted these other theories from consideration.

16 

the EMH is both elegant and simple, there is evidence that practitioners’ beliefs are not congruent with it s tenets. According to Block (1999), of 295 investment professionals surveyed, 63% strongly disagree with the EMH, 34% were neutral, while only 3% agreed with the principals of the EMH. While studies conducted on the EMH prior to the 1980 ’s typically did not distinguish between small and large capitalization stocks, more recent studies have begun to contrast how the EMH holds when considered in the context of different market capitalizations. Panagiotidis (2005) considers small, medium and large capitalization stocks on the Athens Stock Exchange using five statistical tests to examine the residuals of the random walk for the period 1 June 2000 to 14 March 2003 for large capitalization stocks and 1 June 2001 to 12 March 2003 for medium and small capitalization stocks. In all three market capitalization categories the theory of random walk is rejected and no evidence of market efficiency is found. Kalash and Hossien (2008) use the Sharpe ratio to measure risk adjusted returns among small, medium and large capitalization stock mutual funds over the period 1994 to 1997. Their results show that the Sharpe ratios for mutual funds across the spectrum of company capitalization are significantly different and they thus conclude that the mutual fund market is not efficient. Hung, Lee and Pai (2009) consider small and large capitalization stocks listed in Japan and in the United Kingdom. Using parametric and nonparametric variance ratio tests over the period of 1 January 1993 to 17 October 2005 for the Topix stock index and 1 January 1986 to 17 October 2005 for the FTSE stock index, they find support for the weak form of the EMH for large capitalization stocks but reject the hypothesis that weak form efficiency holds for small capitalization stocks. In aggregate, these studies indicate that the theory of the EMH does not always hold for small capitalization stocks. As such, it is expected that this study may show a significant level of predictive power for the TOU and TOD, thus inferring the weak form of market efficiency is not statistically robust. 

2.1.2 The Random Walk Hypothesis 

The EMH relates closely to the theory of random walk in that both assume that the price today is independent from the price of yesterday. At the most basic level the random walk describes the movement in prices from one interval to the next as random deviations from the previous price and infers that the price today is not dependent on the price of yesterday (Malkiel, 2003). Like the EMH, the random walk hypothesis offers no theoretical support for technical analysis. Both the random walk and the efficient market hypothesis have been challenged by market anomalies, including the weekend effect, January effect and the turn of the month effect. Such anomalies highlight repetitive price movements that cannot be explained by either the theory of random walk or the EMH. Cross (1973) shows evidence of the weekend effect by examining the returns on the S&P 500 from 1953-1970 and reports that the index rose on 62% of Fridays and only 39% of Mondays, with a mean return of 0.1% and -0.2%, respectively. The 17 January effect is based on the work of Rozeff and Kinny (1976) who find that over the period 1904-1974 stocks on the NYSE generated average returns in January of around 3.5% compared to average monthly returns in the other months of the year of just 0.5%. The turn of the month effect was first described by Lakonishok and Smidt (1988) using time series analysis on the Dow Jones Inustrual Index for a 90-year period, from 1897 to 1986. They find that monthly returns on the last day of a month and the first three days of the following month are 0.5% compared to an average return for a four day period of only 0.06%. These market anomalies suggest that markets are not random, implying that the two tenets of technical analysis (that prices tend to persist and market action is repetitive) may be relevant in explaining market price movements. If price action is repetitive then it would be a natural progression to assume that repetitive human behavior may play a role in how prices are determined. 

2.1.3 Behavioral Finance 

Behavioral finance attempts to examine financial markets by incorporating observable systematic human behavior (Ricciardi & Simon, 2000). According to behavioral finance, financial markets are not efficient because investor decisions are driven by irrational behavioral heuristics and cognitive biases rather than according to optimal portfolio theory. Heuristics are broadly defined as simple rules, shortcuts or tactics used to solve problems; while cognitive biases are opinionated inclinations or systematic distortions used when making decisions (Buchanan & Huczynski, 2004). One of the central goals of behavioral finance is to understand the systematic market implications of the psychological traits of market players (Stracca, 2004). Hence, behavioral finance marks a significant divide between the investigation of financial market phenomena based on theory that excludes the human psyche and that based on observations that include psychological and behavioral effects. Several findings from the literature support behavioral finance and undermine the tenets of market efficiency. The renowned behavioral economist Richard Thaler (1994) finds that investors are prone to frequent mispricing, as evidenced by mean reverting stock prices that imply rather than trade at a true value, stocks generally trade above and below that value over time (Thaler, 1994). DeBondt and Thaler (1985, 1987) also argue that investors are not rational when making financial decisions and Statman (1999) states that assuming investor rationality and market efficiency is unrealistic because investors are affected by cognitive biases and emotions. Fama and French (1992) find evidence that returns are negatively related to firm size and positively related to the book-to-market ratio. Along similar lines, work conducted by Brennan et al. (1998) shows that investments based on firm size and book-to-market ratios result in reward-to-risk ratios more than twice as high as investing in the market benchmark. Rouwenhorst (1999) reports that book-to-market ratios and the size of the firm can predict returns in emerging markets. In examining variables effecting returns, Haugen and Baker (1996) find that past 18 returns and trading volume are among the most powerful determinants, negating the notion that prices are independent of the past. In examining how investment decisions are impacted by emotional responses to information, several other studies provide evidence that emotions drive investment decisions, a finding that is at odds with the concept of rational agents and an efficient market (Lo, Repin, & Steenbarger, 2005; Lo & Repin, 2002; Steenbarger, 2002). Collectively, these findings are important as they make a case that investors are not always rational and that historical price information can play a role in determining future prices. Thus in summary, while the EMH and the theory of random walk provide elegant qualitative and quantitative theoretical constructs for describing market pricing, they are subject to formidable scrutiny. The body of evidence from behavioral finance academic literature is increasingly diminishing the tenets that prices are independent of the past and that market actors are rational and non-emotional. With the EMH undermined, the theoretical landscape is increasingly open to technical analysis being justifiably useful. The following section of the literature review considers candlestick analysis test results that may offer further perspective on behavioral finance and the EMH. 

## 2.2 Candlestick Analysis 

The purpose of this section is to discuss the studies that investigate the predictive power of candlestick analysis. The primary points of interest include identifying the markets considered; the dates over which the markets were evaluated; the holding strategy employed; the candlestick patterns evaluated and the tests results of those patterns. This sub-section is structured in two parts. First, general candlestick studies that do not investigate the TOU and TOD are reviewed and thereafter, candlestick analysis that consider the TOU and TOD are reviewed. 

2.2.1 General Candlestick Studies 

Developed Markets: 

Fock et al. (2005) consider an intra-day analysis of candlestick patterns by examining twenty candlestick patterns using five-minute candlestick intervals for the German ten-year government bond (Bund) and the German DAX Stock index futures for the period 1 January 2002 to 31 December 2003. Using a T-test and a static holding strategy 8, they find that only the dark cloud cover 9 has moderate statistical significance while 16 of the remaining patterns generate more losing trades than winning trades.    

> 8Static holding strategy infers the holding strategy is to liquidate the stock position in its entirety at the closing price of period n.
> 9The Dark Cloud Cover is a 2-day bearish reversal pattern. This first day is an up day and is part of an up trend, while the second day is a down day that opens above the high of the first day and closes below the midpoint of the first day.

19 Lu (2013) examine twenty-four trend context two-day candlestick patterns for the period 2 January 2003 to 31 October 2012 for the constituents of the FTSE 100 in the UK, the DAX 40 of Germany and the CAC 30 of France, using a ten-day static holding period. Of the twenty-four patterns tested, only three show moderate statistical significance but none show consistent results across markets. Hence, Lu concludes that different candlestick approaches need to be applied in different markets, suggesting the utility of particular candlestick patterns should not be generalized across markets. Such intra-market sensitivity to candlestick patterns could be the result of differing levels of efficiency, different market participants, different behavioral tendencies among investors or the result of other variable factors affecting market pricing. There is however, other evidence pointing to the contrary, indicating continuity between different markets, including the case of the dark cloud cover which showed significance in both developed markets (the Bund and DAX) as shown by Fock et al. (2005), as well as in the emerging market of the Taiwan Stock exchange as noted by (Shiu & Lu, 2011). 

Emerging Markets: 

Shiu and Lu (2011) consider three two-day bearish reversal patterns and three two-day bullish reversal patterns for the period 1998 to 2007 using a quantile regression method for 69 securities traded on the Taiwan Stock Exchange and a static holding period strategy for days one through five. The results show that the bearish engulfing pattern 10 in days three, four and five after the pattern and the bearish harami 11 on days one through five are highly significant, while the bullish harami is moderately significant for day five and the bullish harami and dark cloud cover are only weakly significant on days four and five, respectively. There is thus a degree of similarity between the results of Shiu and Lu (2011) and Fock et al. (2005) who finds moderate statistical significance for the 

dark cloud cover . It is notable that this similarity transcends both the temporal metric and market considered. Fock et al. (2005) considered five-minute interval candlesticks for two developed markets (the Bund and DAX) while Shiu and Lu (2011) considered daily candlesticks for an emerging market (the Taiwan Stock Exchange). Lu et al. (2012) consider the same six reversal patterns as Shiu and Lu (2011) for the period 29 October 2002 to 31 December 2008 for the Taiwan Top 50 constituents using skewness adjusted T-tests for average profits and a binomial test for winning rates. However, Lu et al. ’s (2012) approach is unique in that the trade exit is not based on a temporal metric, but rather the manifestation of a pattern with the opposite sentiment to that which initiated the trade. The results show that there is moderately significant evidence for the three bullish patterns, which   

> 10 The Bearish Engulfing pattern is a 2-day bearish reversal pattern. The first day is an up day and is part of an up trend, while the second day is a down day and has a body that literally engulfs the body of the first day, meaning its open is higher than the first days close and its close is lower than the first days open. A Bullish Engulfing pattern is the opposite formation.
> 11 The Bearish Harami is a 2-day bearish reversal pattern. The first day is an up day and part of an up trend, while the second day is a down day with an open and close that are inside the body of the body of the first day. A Bullish Harami is the opposite formation.

20 contrasts with the results of Shiu and Lu (2011), who found evidence that two of the mutually investigated bearish patterns considered showed high significance, but accords with the finding of weak significance for the bullish harami. Given the stark difference in the exit strategy between the two studies, such similarities are difficult to compare and may be spurious. Lu and Shiu (2012) again consider the Taiwan Top 50 constituents but investigate twenty-four trend context two-day candlestick patterns, including the same six reversal patterns investigated by Lu et al. (2012) and Shiu and Lu (2011). Using skewness adjusted T-tests for average profits and a binomial test for winning rates, they consider the period 1 January 2002 to 31 December 2009 with static holding periods for one, five and ten days. In contrast to Lu et al. (2012) and Shiu and Lu (2011), Lu and Shiu (2012) find that there is no statistically significant evidence for any of the two-day patterns. The disparate results suggest that a difference in the holding period strategies and testing methods impact the results substantially. Lu (2014) examines twelve trend context one-day patterns for the period 4 January 1992 to 31 December 2009 for 151 stocks traded on the Taiwan Stock Exchange using a static ten-day holding period. Of the twelve single line patterns investigated, four are found to be moderately significant. In addition, Lu reports that the candlestick approach results in higher returns for smaller companies with lower prices, compared to larger companies with higher prices, but does not provide evidence to support this claim. In the South American markets, Prado et al. (2013) consider sixteen trend context candlestick patterns based on the work of Morris (2006) . They analyze ten stocks representing more than 40% of the volume traded on the Sao Paulo Stock Exchange for the period 2005 to 2009 and considered the returns over a static holding period for days one through seven. Using a binomial distribution, they determine that of the sixteen patterns considered, there are eight that show moderate statistical significance for at least one of the seven days in the holding period. The results were then compared to those published by Morris (2006). However, there is only one case of consistent results across the markets, where the same pattern leads to a statistically significant result of predictive power. Hence, Prado et al. (2013) conclude that the cross-market applicability of candlestick patterns cannot be assumed and different candlestick trading strategies need to be considered for different market regions. 

Summary: 

A review of the general candlestick studies shows that there is limited consistency of results. In the case of developed markets, studies find that only a few strategies have significant results and the results are inconsistent across different markets. Similarly, with regards to emerging markets, there is a lack in the consistency of results across time and across markets. 21 Furthermore, studies devoted to both emerging and developed markets have used varied testing procedures, time periods and holding periods and thus it is not unexpected that there is evidence that both negates and supports the notion of continuity of candlestick returns across time and markets. 

Table 1: Summary of General Candlestick Studies 

Author(s) Interval / Market Dates 

Holding 

Strategy 

TOU / 

TOD Result* 

Fock et al . 2005 5 minute Intraday / DAX 

& Bund Futures 

1 Jan 20 02 – 31 

Dec 20 03 

Static, 30 

minutes 

No Negative 

Shiu, Lu 2011 Daily / Taiwan Stock 

Exchange, 69 stocks 

1998 – 2007 Static, Days 

1-5

No Negative 

Lu et al. 2012 Daily / Taiwan Top 50 29 Oct 20 02 –

31 Dec 20 08 

Static, 

NA* *

No Positive 

Lu, Shiu 2012 Daily / Taiwan Top 50 01 Jan 20 02 –

31 Dec 20 09 

Static, Days 

1,5,10 

No Negative 

Lu, 2013 Daily / FTSE 100, DAX 

40, CAC 30 

2 Jan 20 03 – 31 

Oct 20 12 

Static, Day 

10 

No Negative 

Prado, 2013 Daily / Brazil Sao Paulo 

Exchange, 1 0 stocks 

2005 – 2009 Static, Days 

1-7

No Negative 

Lu, 2014 Daily / Taiwan Stock 

Exchange, 151 stocks 

4 Jan 19 92 – 31 

Dec 20 09 

Static, Day 

10 

No Negative   

> * Where the TOU / TOD were not tested, Positive result infers more than half of the results for all the patterns tested generated a positive return. While this is an effort to simplify overall results it bears no indication of statistical significance. **Lu et al. (2012) apply a holding strategy that liquidates the position only when a candlestick with an opposing sentiment manifests. The average duration of the time period was not disclosed.

2.2.2 Candlestick Studies that Consider the TOU and TOD 

Developed Markets: 

Caginalp and Laurent (1998) are among the first to employ a scientifically robust testing method for candlestick analysis to investigate eight trend context reversal patterns, including the TOU and TOD, on the constituents of the S&P 500 for the period 19 January 1992 to 14 June 1996. Using the normal approximation to the binomial 22 method and testing four different dynamic holding strategies 12 , they find that all of the patterns have moderately significant predictive power. The results support the efficacy of the predictive power of candlestick analysis by showing consistent results for the patterns tested. Thus the results also contest the theory of market efficiency in one of the world’s largest equity markets. 

Marshall et al. (2006) examine fourteen trend context single line patterns and fourteen trend context reversal patterns, including the TOU and TOD for the constituents of the Dow Jones Industrial Index for the period 1 January 1992 to 31 December 2002. Employing a bootstrap methodology and a 10-day static holding period, they find none of the patterns have significant predictive power. With regard to the TOU and TOD, they report that they are profitable 48.39% and 48.89% of the time, respectively. Marshall et al. (2006) thus concludes that their findings are consistent with market efficiency and using candlestick analysis does not add value for investors trading large capitalization stocks in the US market. Horton (2009) examines the same eight bullish and bearish trend context reversal patterns considered by Caginalp and Laurent (1998) and Lu et al. (2015), for 349 stocks from the Value Line database for an undisclosed time period. Horton employs three nonparametric tests (the Kolmogorov –Smirnov test, the Cramer –Von Mises test, and the Birnbaum –Hall test). The results show no meaningful evidence of predictive power and Horton concludes that the candlestick patterns are not recommended for investors. Duvinage et al. (2013) investigates the Dow Jones Industrial Index using 5-minute intervals and examines eighty-three trend context candlestick patterns, including the TOU and TOD over the period 1 April 2010 to 13 April 2011, using a static holding period of 50 minutes. While statistical testing on mean returns and Sharpe ratios indicates that 26 of 83 patterns are weakly significant, after applying filters for risk and trading costs, they conclude that none of the patterns outperform a buy and hold strategy. These results are consistent with the other investigation into intraday candlestick analysis by Fock et al. (2005) who tested twenty patterns using 5-minute intervals and a 30 minute static holding period on the DAX and Bund futures, but found no statistical evidence of predictive power, with only one exception of the dark cloud cover pattern. 

Emerging Markets: 

Goo et al. (2007) investigate twelve single line and fourteen trend context reversal candlestick patterns for the period 1997 through 2006 for the 25 highest market share equities in Taiwan. Applying T-tests to analyze profitability and ANOVA and Duncan's multiple range tests to examine a General Linear Model, they compare the     

> 12 Dynamic holding strategy infers the holding strategy is to liquidate the position at the closing price on days X, X+1, X+2, X+n after the patterns manifest, in equal proportions, at the closing price. For instance a 3-day dynamic holding strategy would liquidate a 30 share position by selling 10 shares at the closing price on 1 st , 2 nd and 3 rd day after the pattern manifests.

23 profitability of candlesticks on different holding days. The results show that for days one through ten, there is a moderately significant and positive relationship between the holding period and the return generated for all patterns. In addition, the positive returns for the TOD are significant only on day three and for the TOU on days one, seven, eight, nine and ten, all at the 5% significance level. Thus the results suggest a difference in the utility of the TOU and TOD for large capitalization stocks in Taiwan versus the United States, where Marshal et al. (2006) found no evidence of predictive power. Marshall et al. (2007) examine fourteen single line and fourteen trend context reversal patterns for 100 medium and large capitalization stocks listed on the Japan Topix 70 Index and the Topix Mid 400 Index for the period 1975 to 2004. Similar to Marshall et al. (2006), they employed a bootstrap methodology to consider two, five and ten-day static holding periods. Once again Marshall et al. (2007) find that even before transaction commissions, candlestick patterns create no value for investors. They reveal that over a ten-day holding period the TOU and TOD show profits 44% and 56% of the time, respectively, and that over the two, five and ten-day holding periods, the TOU and TOD are in the top decile of overall performance. The results infer consistency in the findings of Marshall et al (2007, 2006) in that across both emerging and developed markets there is no statistically significant support for the predictive power for the TOU and TOD. Lu et al. (2015) investigate the same eight trend context three-day reversal patterns examined by Caginalp and Laurent (1998) and Horton (2009) for the period 2 January 1992 through December 31, 2012 for constituents of the Dow Jones Industrial Index. Using T-tests and Step-SPA tests, they include three different trends and four different holding strategies and investigate differences in how the trend and holding strategy impact returns. They find that regardless of the trend definition, all of the patterns are profitable at a moderate level of significance after accounting for transaction costs when using a three-day dynamic holding period strategy. Lu et al. (2015) also show that holding strategy plays a critical role in determining successful trades. The first holding strategy they consider is the dynamic holding strategy employed by Caginalp and Laurent (1998) in which a trade is exited in proportions equal to one divided by the number of days in the holding period, at the close of each day of the holding period. Applying this approach over a three-day and ten-day period they find all eight reversal patterns earn moderate statistically significant positive returns across the three trend definitions with only two exceptions in the 48 permutations of holding period and trend, after accounting for a 0.5% transaction cost. When the static holding strategy is analyzed, in which the trade is simply exited on day-three or day-ten at the closing price, they find all the patterns fail to produce significant returns for both holding periods, with one exception in the 48 permutations of holding period and trend, after accounting for a 0.5% transaction cost. 24 

Summary: 

Similar to the results of the general candlestick studies, candlestick studies that explicitly investigate the TOU and TOD show mixed results. There is more consistency in the results based on the holding period strategy than there is with the patterns examined. In both the developed markets and emerging markets, studies that used a static holding period strategy show negative results while those that used a dynamic holding period strategy show positive results. This relationship is confirmed by Lu et al. (2015) by directly comparing the two holding period strategies for the same data-set and finding analogous outcomes, that is, significant returns when using the dynamic holding period strategy but no significant returns when using a static holding period strategy. 

Table 2: Summary of Candlestick Studies that consider TOU and TOD Events 

Author(s) Interval / Market Dates 

Holding 

Strategy 

TOU / 

TOD Result *

Caginalp & 

Laurent, 1998 

Daily / S&P 500 

constituents 

19 Jan 19 92 – 14 

June 19 96 

Dynamic, 

Days 1,2,3; 

1,2; 2,3; 2,3,4

Yes Positive 

Marshall et al., 

2006 

Daily / DJIA 

constituents 

1 Jan 19 92 – 31 

Dec 20 02 

Static, Day 10 Yes Negative 

Goo et al, 2007 Daily / Taiwan Stock 

Exchange, 25 stocks 

1997 – 2006 Static, Days 

1-10 

Yes Negative 

Marshall et al. 

2007 

Daily / Japan Topix 70 

& Topix Mid 400 & 100 

1975 – 2004 Static, Days 

2,5,10 

Yes Negative 

Horton, 2009 Daily / USA Value Line 

Database 

Not Di sclosed Static, Days 

1,2,3 

Yes Negative 

Duvinage et al., 

2013 

Intraday 5-minute /

DJIA Constituents 

1 April 20 10 – 13 

April 20 11 

Static, 50 

minutes 

Yes Negative 

Lu et al., 2015 Daily / DJIA 

Constituents 

2 Jan 19 92 – 31 

Dec 20 12 

Static , Days 3, 

10 . Dynam ic 

Days 1,2,3; 1-

10 

Yes Positive      

> *Where the TOU / TOD were tested, Positive result infers the TOU and TOD both generated positive returns where positive returns are achieved when the proportion of returns greater than 0 is greater than .5. If the results are evenly split, to be conservative, they will be marked as being Negative . While this is an effort to simplify overall results it bears no indication of statistical significance.

25 

## 2.3 Conclusion 

The discussion of theory relating to candlestick analysis indicates a division between theory that assumes rational actors and theory that posits non rational actors with whom the human psyche plays a role in dictating market prices. The weak form of the EMH posits that all past information is effectively priced into a stock, thus excluding candlestick analysis as a viable study (Fama, 1970). The majority of practitioners however, strongly disagree with the EMH (Block, 1999) and there is empirical evidence that small capitalization stocks are less efficient than their larger capitalization counterparts (Hossein & Kalash, 2008; Hung et al., 2009), suggesting kinks in the EMH. Like the EMH, the theory of random walk (which posits that future prices are independent of current prices), also implies that past price information is of no utility for the investor and thus excludes the possibility that candlestick analysis could be useful. Behavioral finance, on the other hand, argues that market participants are not rational and are thus prone to irrational behavior brought on by psychological heuristics and biases. As such, behavioral finance poses no theoretical limitations on the tenets of technical analysis and does not restrict it as a viable investment strategy. Nison and Morris introduced candlestick analysis into Western markets in 1991 and 1992, respectively. They both emphasized that for a reversal pattern to have predictive power, a trend must be in place. Lu et al. (2015) show evidence indicating the type of trend definition does not have a material impact on returns but the holding period strategy does. The literature considered herein supports this finding and while the results of investigation into the TOU and TOD are mixed, the evidence of a positive relationship between a dynamic holding period strategy and statistically significant returns is strong, albeit supported by a small sample of studies. Of the five studies that consider the TOU and TOD using a static holding period strategy, none of the results are statistically significant. Of the two studies that consider the TOU and TOD using a dynamic holding period strategy, both generate statistically significant results. To date, no candlestick study has investigated small capitalization stocks. Among the stocks considered are the constituents of S&P 500, the Dow Jones Industrial Index, the Japan Topix 70, Topix Mid 400, Topix 100, the FTSE 100, the DAX 40, the CAC 30, the Taiwan Top 50, the 25 highest market capitalization stocks of the Taiwan Stock Exchange, 151 stocks on the Taiwan Stock Exchange and the 10 most liquid stocks on the Sao Paulo Exchange in Brazil. Since the seminal work of Caginalp and Laurent (1998) there have been less than eight empirical investigations into the TOU and TOD, none of which considered small capitalization stocks and none of which considered a period that extended past 2012. Moreover, only two of the works utilized a dynamic holding strategy to measure investment returns. 26 As will be detailed in the Research Methodology section below, this study seeks to fill this gap in the literature by investigating three dynamic holding period strategies for the TOU and TOD for small capitalization stocks in the S&P 600 for the period 1 June 2005 to 15 May 2015. 27 

# Chapter 3 - Research Methodology 

## 3.1 Research Design 

This study uses a deductive approach, which begins with generalizations and aims to determine if the generalizations are applicable to particular instances (Hyde, 2000). The analysis herein makes use of a longitudinal investigation in which quantitative time series data is gathered, analyzed and used to test a hypothesis. The analysis techniques employed are based on the seminal work of Caginalp and Laurent (1998) who utilize the normal approximation to the binomial method. The analysis of the data is conducted with Microsoft Excel, which is the most appropriate tool given the need for pattern recognition formulas, data aggregation, filtering, usability and cost constraints. Due to the relative simplicity of the normal approximation to the binomial method, standalone statistical software was deemed unnecessary. 

## 3.2 Data Collection & Sample 

3.2.1 Data Collection 

In order to consider the hypothesis herein, data was collected for the daily opening and closing prices for 47 small capitalization stocks in the USA over the 10 year period from the 1 st of June 2005 to the 15 th of May 2015. The data was sourced from Reuters DataStream , which is one of the two prominent subscription based providers of financial data to institutional investors and investment banks globally. The use of daily opening and closing price data over a 10 year period enables the analysis to cover both bull and bear market conditions and provides a sufficient number of data points to address the hypothesis robustly. 

3.2.2 Sample 

To provide a representation of small capitalization stocks, the sample is comprised of 47 stocks that trade on the NASDAQ and NYSE in the USA and are listed constituents of the S&P 600 index. The sample is composed only of those stocks that as of 1 st June 2005 were constituents of the S&P 600 and had a market capitalization of 1 billion USD or higher and as of 15 th May 2015 still remained in the S&P 600 index. 13  

> 13 See Appendix I to view the sample list.

28 The focus on small capitalization stocks in the USA is driven by the gap in the literature. Presently there are no other studies that examine small capitalization stocks in the context of candlestick analysis. The rationale for selecting small capitalization stocks from the S&P 600 is based on this index providing a deep representation of small capitalization stocks in the USA. To be part of the S&P 600 stocks must meet the following criteria (S&PDowJones, 2014): They are US companies; they have an unadjusted market capitalization of USD 400 million to USD 1.8 billion upon entry into the index; at least 50% of the shares outstanding are available for trading; the companies have positive earnings in the most recent quarter, as well as over the most recent four quarters; and the stocks are liquid with active secondary markets on the NYSE or NASDAQ stock exchanges. In addition, the use of a sample collected from the S&P 600 accomplishes the following four research goals: 1. Transparency of stock selection criteria is established and the selection criteria is replicable. 2. Temporal continuity is achieved, simplifying the analysis. 3. The stocks considered are liquid and meet key investor criteria. 4. In addition to the three criteria above, by choosing only those stocks with a market capitalization exceeding 1 billion USD in 2005, the concern expressed by Marshall et al. (2006) (that small companies are not worth considering because of liquidity limitations) is addressed. 

## 3.2 Data Analysis Methods 

To address the question of the predictive power of the TOU and TOD, two steps were required: 1. A determination of a robust statistical method to judge predictive power (see 3.2.2 below) 2. The creation of an analytical model to judge predictive power (see 3.2.3 below) 

3.2.1 Data and Naming Conventions 

The two data conventions applicable to this study are the day count convention and the open-close convention. The day count convention is defined on the basis that Tn represents the nth day of the TOU or TOD. Hence, T1 is the first day of the three-day TOU or TOD, T2 the second and T3 is the last day of the pattern. T4 is the first day after the TOU or TOD while T0 is the day before the first day of the pattern. The open (O) and close (C) convention is denoted as C1 for the closing price on day T1, C2 for the closing price on day T2 and so on. In the above sections, the names TOU and TOD have been used to describe the trend context three-day TOU and TOD patterns (that is, when the TOU / TOD manifest in the context of a down / up trend). From here on, for analytical purposes, the two patterns will be deconstructed into the TOU, TOU Event, TOD and TOD Event, 29 respectively. The TOU and TOD names will represent the three-day patterns in isolation, while the TOU Event and TOD Event names will represent the three-day patterns when they have manifested in a down trend and up trend, respectively. 

3.2.2 The Normal Approximation to the Binomial 

To measure the predictive power of the TOU Event 14 , the normal approximation to the binomial method is utilized, based on the seminal work of Caginalp and Laurent (1998). The normal approximation to the binomial method allows for the comparison of the means of two samples to determine the extent of their difference. In the context of this analysis, the mean of a sample is defined as the proportion of successes (positive returns) from the TOU Event within the sample period. The methodology will allow for the comparison of the proportion of successes of the TOU Event to be compared to a base case - where the base case is the proportion of successes after a down trend occurs. Comparing the proportion of successes of the TOU Event to the proportion of successes from a down trend, accounts for the possibility that price action after a TOU Event is a function of the trend rather than the TOU Event itself. The base case comparison thus affords the capability to distinguish the return impact of a down trend from the return impact of a TOU Event by comparing their respective means. In the case of the TOU Event, the return is a success when on an ex-post basis, the opening price on the day after the TOU Event (T4 Open) is observed to be less than the average of the closing prices for the holding period being considered. This infers that after the TOU Event, prices increased for the holding period. Success Open Holding period TOU Event = T4 < Ave. Closing Price 

(1) The mean for the TOU Event is the proportion of TOU Event successes in the sample and is denoted as p where: Success TOU Event p = n

(2) where n is the number of TOU Events in the sample. The price change after a down trend is measured in the same way as for a TOU Event, by the difference between the opening price on the day after the trend is identified and the average of the closing prices for the holding period. A down trend success is defined as occurring when, on an ex-post basis, the open price on the day after the trend event is observed to be less than the average of the closing prices for the holding period.  

> 14 For explanatory clarity, only the TOU Event will be detailed in this section. The principles that apply for the TOU Event also apply for the TOD Event, bearing in mind that the TOD Event is the inverse of the TOU Event.

30 Success Open Holding period Down Trend = T4 < Ave. Closing Price 

(3) The mean for the down trend is the proportion of down trend successes and is the theoretical mean of the binomial distribution and is denoted as P0 where: Success Down Trend P0 = N

(4) where N is the number of down trends in the sample. If the TOU Event bears no predictive power then p = P0 .Hence, the null and alternative hypotheses can be defined as follows: H0: The proportion of TOU Event Success for the sample of 47 small capitalization stocks drawn from the S&P 600 for the period 1 June 2005 to 15 May 2015, is less than or equal to the proportion of Base Case Successes at a 95% confidence level. H0: p P0 

H1: The proportion of TOU Event Success for the sample of 47 small capitalization stocks drawn from the S&P 600 for the period 1 June 2005 to 15 May 2015, is greater than the proportion of Base Case Successes at a 95% confidence level. H1: p P0 

The decision rule to reject or not reject the null hypothesis is based on the Z-statistic, in accordance with Caginalp and Laurent (1998). The Z-statistic is derived from the following formulas (Caginalp & Laurent, 1998): Variance = nP0(1-P0) 

(5) Standard Deviation = nP0(1-P0) 

(6) n(p-P0) Z = nP0(1-P0) 

(7) The normal approximation to the binomial method and the large sample size imply that the normal distribution can be assumed in accordance with the central limit theorem. Accordingly, if the derived Z-statistic is greater than the critical 95% confidence level (Z-value of 1.65) then the null hypothesis can be rejected. As such, the decision rule can be stated as follows: Decision Rule: Reject Ho if Z-statistic > 1.65 31 

3.2.3 Modeling the Normal Approximation to the Binomial 

The normal approximation to the binomial method requires four key inputs, the number of TOU Events ( n), the proportion of TOU Event successes ( p), the number of down trends ( N) and the proportion of down trend successes ( P0 ). Figure 4 summarizes the 7-step process used to generate p and n.15 

Figure 4: Seven-Step Data Manipulation for Inputs p and n

Simple Three-Day Moving Average: 

The calculation for the three-day simple moving average is calculated as: t-2 t-1 t[ C + C + C ] MA(3)= 3

(8) 

Trend Identification: 

Nison (1991) and Morris (1992) emphasize that the formation of a trend is crucial for identifying reversal patterns because a reversal pattern can only be validated in the context of a trend. Hence, defining a trend is an integral part of robust candlestick analysis but there is little agreement on the most robust trend metric. Caginalp and Laurent (1998) and Horton (2009) employ the three-day moving average to define an up trend (down trend) as occurring when the three-day moving average has increased (decreased) on six consecutive days, with the permission of one deviation from this rule. Marshall et al. (2006 2007) use a ten-day exponential moving average to define and an up trend (down trend) as occurring when the price is above (below) the ten-day exponential moving average. Goo et al. (2007) use a five-day moving average and define an up trend (down trend) as occurring when the five-day moving average has increased (decreased) on six consecutive days. Lu et al. (2015) consider three different trend definitions, including the Levy Trend which is derived from the slope of a linear formula; a      

> 15 For explanatory clarity, in detailing the derivation of pand n, only the TOU Event is considered. The principles that apply for the TOU Event also apply for the TOD Event, bearing in mind that the TOD Event is the inverse of the TOU Event.
> 1

• Simple 3 -Day Moving Average  

> 2

• Trend Identification  

> 3

• TOU Identification  

> 4

• TOU Event Identification  

> 5

• TOU Event Return Calculation  

> 6

• TOU Event Return Aggregation  

> 7

• Calculation of p & n 32 three-day moving average to define an up trend (down trend) as occurring when the three-day moving average has increased (decreased) on six consecutive days; and an exponential moving average of three and ten-days. Lu et al. (2015) find the definition of trend does not impact the performance of the candlestick patterns they tested, which included the TOU and TOD Events. This study makes use of a three-day simple moving average and a down trend (up trend) is defined as occurring when the three-day simple moving average decreases (increases) on six consecutive days. According to the analysis of the data, there are 9,290 down trend events, as defined by equation (9): t t-1 t-2 t-7 Down Trend = MA(3) <MA(3) <MA(3) <...MA(3) 

(9) 

TOU Identification: 

The three-day TOU pattern is identified using the following sequence of inequalities: Day 1 is a down day if C1 < O1 Day 2 opens lower than day 1’s close if O2 < C1 Day 2 closes above day 1’s open if C2 > O1 Day 3 closes above d ay 2’s close if C3 > C2 Day 3 is an up day if C3 > O3 

Day 3’s open is greater than day 2’s open if O3 > O2 

TOU Event Identification: 

A TOU Event is identified when the TOU three-day pattern manifests in a downtrend. This method identified 177 TOU Events. 

TOU Event Return Calculation: 

To capture a broad spectrum of returns, this study considers three different holding periods that span a nine day period following the TOU Event. The first, (Period 1) considers a three day holding period that involves selling a third of the stock each day at the closing price in days T4, T5 and T6 (the first, second and third day after the pattern event, respectively). The second (Period 2) considers a three day holding period that involves selling a third of the stock each day at the closing price in days T7, T8 and T9 (the fourth, fifth and sixth first day after the 33 pattern event, respectively). Similarly, the third (Period 3) considers a three day holding that involves selling a third of the stock each day at the closing price in days T10, T11 and T12 (the seventh, eighth and ninth day after the pattern event respectively). 16 

Thus the three holding period returns can be defined as: Close Close Close Open Return1 Open 

T4 +T5 +T6 -T4 3TOU Event T4 

      

(10) Close Close Close Open Return2 Open 

T7 +T8 +T9 -T4 3TOU Event T4 

      

(11) Close Close Close Open Return3 Open 

T10 +T11 +T12 -T4 3TOU Event T4 

      

(12) 

TOU Event Return Aggregation and Calculation of p and n: 

The list of percentage price changes calculated using equations (10), (11) and (12) are then used to extract the aggregate return data and calculate the value for p. If the returns from these equations are positive, a TOU Event success is counted. The calculation of n is simply a matter of summing the TOU Events. 

Calculation of P0 and N: 

The same logic that applied to p and n is also used to determine P0 and N. The three-day moving average and trend identification are defined as before in equations (8) and (9) and the trend return is calculated for the same three holding periods as above, using the following formulas:  

> 16 The implicit assumption in these holding periods is that the stock bought could be sold without market friction at the closing price of each day. This assumption and the convention of entering a trade at the opening price the day after the pattern event (T4) is consistent within the literature.

34         

> Close Close Close Open Return1 Open

T4 +T5 +T6 -T4 3Down Trend T4 

(13)         

> Close Close Close Open Return2 Open

T7 +T8 +T9 -T4 3Down Trend T4 

(14)         

> Close Close Close Open Return3 Open

T10 +T11 +T12 -T4 3Down Trend T4 

(15) The list of percentage price changes calculated using equations (13), (14) and (15) are then used to extract the aggregate return data and calculate the value for P0. If the returns from these equations are positive, a down trend success is counted. The calculation of N is then simply a matter of summing the down trends. 35 

# Chapter 4 - Research Findings & Analysis 

The purposes of this chapter is to review and discuss the outcomes of the aforementioned tests on the predictive power of the TOU Event and TOD Event. For clarity, the TOU Event and the TOD Event are discussed separately. 

## 4.1 TOU Event Findings and Analysis 

To illustrate the use of the normal approximation to the binomial method, the case of the TOU Event over Holding Period 1 is first discussed. For the stocks under consideration, there are 9,290 down trend events, of which 4,887 generate positive returns (base case success). 177 TOU Events are observed, of which 86 generate positive returns (TOU Event success). As can be seen from Table 5 below, the expected number of successes ( nP0 ) is 93 while the actual number of successes is 86, lower than expected. The Z-score of -1.07 is less than the critical score of 1.65 and thus the null hypothesis that the TOU Event does not have predictive power for small capitalization stocks within the S&P 600 for the period 1 June 2005 to 15 May 2005, cannot be rejected. Table 5 also shows that the Z-scores for Holding Period 2 and Holding Period 3 are similarly, below the critical level of 1.65 and thus it is not possible to reject the null hypothesis that the proportion of TOU Event successes for the sample is less than or equal to the proportion of base case successes. Hence, there is no evidence to substantiate a claim that the TOU Event has predictive power to generate positive returns for any of the holding periods considered. The proportion of down trend successes ( PO ) is 0.526, 0.536, and 0.521 for Holding Period 1, 2 and 3, respectively, whereas the proportion of TOU Event successes ( p) is 0.486, 0.475 and 0.463 for Holding Period 1, 2 and 3, respectively. These results indicate that for every holding period, the proportion of success from a down trend is higher than from the TOU Event. The TOU Event generates a negative return more often than a positive return while the down trend generates a positive return more often than a negative return. Moreover, in each holding period, the number of TOU Event successes predicted by the binomial model ( nP0 ) is greater than the value obtained from the sample ( np ). For holding periods 1, 2 and 3 the expected values ( nP0 ) are 93.11, 94.81 and 92.28, respectively, but the actual number of respective successes ( np ) is 86, 84 and 82. These results infer that that the TOU Event has greater capability to predict that prices will continue to go down, rather than reverse and go up, which is the opposite of what the technique is supposed to do according to Morris (1992). That the Z-values determined for Holding Periods 1, 2 and 3 are -1.07, -1.63 and -1.55, respectively, indicates that the results are not far from meeting the threshold that would allow one to reject the opposite null hypothesis -that the TOU Event does not yield predictive power for prices to go down, at the 5% significance level. Using a 36 10% level of significance instead, the critical Z-Value of -1.28 would be breached for Holding Period 2 and Holding Period 3, inferring that at a weak level of statistical significance, the TOU Event does have predictive power for prices to go down for these holding periods. If commissions, taxes, and transaction slippage were to be considered, the results would be even more compelling against the predictive power of the TOU Event. Without a charge for transaction costs, of the 177 TOU Events for Holding Period 1, 2 and 3 there were 86, 84 and 82 successes, respectively. Of those, 48, 70 and 69 of the successes for Holding Periods 1, 2 and 3 generate returns in excess of 1%, which is the most common commission charge applied in the literature (Lu, Shiu, & Liu, 2012; Lu, 2014; Shiu & Lu, 2011). After accounting for a 1% commission charge, the proportions of success for the TOU Event would decrease to 27.1% from 48.6%, to 39.5% from 47.5% and to 38.9% from 46.3% for Holding Period 1, 2 and 3, respectively. This exemplifies the degree to which transaction frictions would further erode the results. When compared to other candlestick studies that also investigate the TOU Event with a dynamic holding period strategy, these results contrast. While this study finds the proportion of profitable trades ( p) for the TOU Event for Holding Period 1, 2 and 3 are 48.6%, 47.5% and 46.3%, respectively, Caginalp and Laurent (1998) found the proportion of profitable trades for the TOU Event to be 53.84% for all four dynamic holding periods investigated, for the constituents of the S&P 500. The more recent work of Lu et al. (2015), in examining the constituents of the Dow 30, found the proportion of profitable trades for the TOU Event to be 69.89% and 66.67% for a 3-day and 10-day dynamic holding period, respectively. While the cause for the differences is beyond the scope of this study, the work of Lu et al. (2015) concludes that the trend definition does not impact the statistical significance of candlestick returns (as long as some form of trend is in place). If that conclusion is assumed to be true, then stock constituents, the time period, the details of the dynamic holding period, intermarket volatility and the testing procedure may all play a role in influencing the dispersion of the aforementioned results. Nonetheless, it is notable that in considering the herein sample of small capitalization stocks, the proportion of successes for the TOU Event is lower than in both cases of comparable studies that also utilized a dynamic holding period strategy, but for large capitalization stocks. 37 

Table 5: TOU Event Holding Period Results Down Trends Down Trend Successes % Down Trend Success TOU Events TOU Event Successes % TOU Event Success TOU Event Expected Successes Z - Score N NP0 P0 N np p nP0 ZTOU Event Holding Period 1 

9,290 4,887 0.526 177 86 0.48 6 93.11 1 -1.07 0

TOU Event Holding Period 2 

9,286 * 4,974 0.536 177 84 0.475 94.809 -1.629 

TOU Event Holding Period 3 

9,260 * 4,828 0.521 177 82 0.463 92.28 5 -1.548 

> *The decrease in N, the number of trend events is a result of the fact that the holding period returns for the trend events consider the days following the trend event. Holding Period 1 considers days 1, 2 and 3 after the trend event, Holding Period 2 considers days 4, 5 and 6 after the trend event and Holding Period 3 considers days 7, 8 and 9 after the trend event. 15 May 2015 is the last day of the data considered in this study; as such the returns for trend events for Holding Period 1 could not be considered after 12 May 2015; the trend events for Holding Period 2 could not be considered after 7 May 2015; and the trend events for Holding Period 3 could not be considered after 4 May 2015. This results in a small discrepancy in the number of trend events counted for each holding period. The discrepancy is less than .323% of the total trend events and is considered acceptable because it has no material effect on the testing procedure or the results.

In conclusion, the results show that there is no statistically significant empirical evidence that the TOU Event has predictive power. That the proportion of successes ( p), is less than 0.5 for all holding periods indicates that more often than not the TOU Event leads to a negative return. The Z-scores are less than 1.65 and closer to -1.65, indicating that the TOU Event comes close to generating negative returns at a statistical significance level of 95%; and at the 10% weak level of significance, generates negative returns for Holding Period 2 and Holding Period 3. Thus, these results indicate that an investor faced with an option to buy after a down trend or buy after a TOU Event would be better off buying after the down trend. These results also infer that the TOU Event is less useful than a down trend in predicting an upturn in prices and does not provide the investor with useful information concerning the reversal of a trend. Hence in summary, at a moderate level of significance, there is no evidence that the TOU Event contests the tenets the Efficient Market Hypothesis. 38 

## 4.2 TOD Event Findings and Analysis 

The TOD Event results are summarized in Table 6 below. As can be seen, the Z-scores indicate that the null hypothesis that the proportion of TOD Events for the sample of 47 small capitalization stocks drawn from the S&P 600 for the period 1 June 2005 to 15 May 2015 is less than or equal to the proportion of Base Case Successes with a 95% confidence level, cannot be rejected. Hence, there is no evidence to substantiate a claim that the TOD Event has predictive power to generate positive returns for any of the holding periods considered. The proportion of down trend successes ( PO ) is 0.499, 0.505 and 0.509 for holding periods 1, 2 and 3, respectively, whereas the proportion of TOD Event successes ( p) is 0.498, 0.484 and 0.480 for holding periods 1, 2 and 3, respectively. These results show that there is no difference between the price action that follows an up trend and that following a TOD Event, for Holding Period 1. Similarly, for Holding Periods 2 and 3, the values differ only slightly (50.5% versus 48.4% and 50.9% versus 48.0% for the up trend and TOD Events, respectively). In each holding period the number of TOD Event successes predicted by the binomial model ( nP0 ) is relatively close to the value that manifested from the sample ( np ). For holding period 1, 2 and 3 the expected values ( nP0 )are equal to 110.31, 111.62 and 112.51, respectively, compared to the actual number of successes ( np ) of 110, 107 and 106. The small difference between the expected values ( nP0 ) and the actual values ( np ), indicates that the TOD Event does not differentiate itself from an up trend in its ability to predict a down turn in prices. Hence, as is the case with the TOU Event, in each holding period the proportion of successes from the trend is higher than the proportion of successes from the TOD Event, resulting in negative Z-scores for each of the holding periods. The finding that the np values are marginally less than the nP0 values is reflected in the low and negative Z-scores of -0.04, -0.62, and -0.88 for Holding Periods 1, 2 and 3, respectively. That none of the Z-scores come close to the critical Z-value of 1.65 infers that the TOD Event does not have predictive power at any significance level. While the values of the proportion of TOD Event successes ( p) do not deviate from the values of the proportion of up trend successes ( P0 ) to the same extent as for the TOU Event, it remains that the TOD Event generates a positive return less than half of the time. Furthermore, the TOD Event generates a positive return less frequently than up trends in all holding periods. This result suggests that the TOD Event is less useful than an up trend in predicting a down turn in prices and does not provide the investor with useful information concerning the reversal of a trend. As with the TOU Event, if commissions, taxes, and transaction slippage were to be considered, the results would be even more compelling against the predictive power of the TOD Event. Without a charge for transaction costs, of the 221 TOD Events for Holding Period 1, 2 and 3 there were 110, 107 and 106 successes, respectively. Of those, 39 64, 85, and 94 of the successes generate returns in excess of a 1% commission charge, for Holding Periods 1, 2 and 3, respectively. After accounting for this 1% charge, the proportions of success (p) for the TOD Event would decrease to 28.96% from 49.8%, to 38.5.1% from 48.4% and to 42.5% from 48.0% for Holding Period 1, 2 and 3, respectively. This shows the extent to which transaction frictions would further erode the case for TOD Event predictive power. When compared to other candlestick studies that investigated the TOD Event with a dynamic holding period strategy, the results determined herein show dissimilarity. This study observes the proportion of profitable trades (p) for the TOD Event to be 49.8%, 48.4% and 48.0% for Holding Period 1, 2 and 3, respectively. Caginalp and Laurent (1998), who examined the S&P 500, found the proportion of profitable trades for the TOD Event to be 76.92%, 73.07%, 69.23% and 76.96% for the four dynamic holding periods they investigated. Lu et al. (2015) examined the Dow 30 constituents and found the proportion of profitable trades for the TOD Event to be 64.29%, and 57.94% for the two dynamic holding periods they investigated. As with the results of TOU Event, the stock constituents, the time period, the details of the dynamic holding period, intermarket volatility and the testing procedure may all play a role in influencing the lack of continuity in the results between these studies. All the same, it ought to be noted that that in examining the herein sample of small capitalization stocks, the proportion of successes for the TOD Event is lower than in the cases of similar studies that also utilized a dynamic holding period strategy, but with large capitalization stocks. 

Table 6: TOD Event Holding Period Results 

Up Trend s

Up Trend 

Successes 

% Up 

Trend 

Success 

TOD 

Events 

TOD 

Event 

Successes 

% TOD 

Event 

Success 

TOD Event 

Expected 

Successes Z - Score 

N NP0 P0 n np p nP0 Z

TOD Even t Holding Period 1 

11,163 5,572 .499 221 110 .498 110.312 -.042 

TOD Event Holding Period 2 

11,157 * 5,635 .505 221 107 .484 111.619 -.621 

TOD Event Holding Period 3 

11,157 * 5,680 .509 221 106 .480 112.511 -.876 

> *The decrease in N the number of trend events is a result of the fact that the holding period returns for the trend events consider the days following the trend event. Holding Period 1 considers days 1, 2 and 3 after the trend event, Holding Period 2 considers days 4, 5 and 6 after the trend event, and Holding Period 3 considers days 7, 8 and 9 after the trend event. 15 May 2015 is the last day of the data considered in this study; as such the returns for trend events for Holding Period 1 could not be considered after 12 May 2015; the trend events for Holding Period 2 could not be considered after 7 May 2015; and the trend events for Holding Period 3 could not be considered after 4 May 2015. This results in a small discrepancy in the number of trend events for each holding period. The discrepancy is less than .054% and is considered acceptable because it has no material effect on the testing procedure or the results.

40 In conclusion, there is empirical evidence that the TOD Event has no significant predictive power. In all of the holding periods considered, the Z-value is negative, and considerably less than the critical Z-value of 1.65 and thus the null hypothesis that the proportion of TOD Event Successes for the sample of 47 small capitalization stocks drawn from the S&P 600 for the period 1 June 2005 to 15 May 2015, is less than or equal to the proportion of Base Case Successes at a 95% confidence level, cannot be rejected. In addition, the proportion of successes ( p) is less than 0.5 for all holding periods, indicating that more often than not the TOD Event results in a negative return. The evidence suggests that an investor faced with an option to sell after an up trend event or sell after a TOD Event, ought to be generally indifferent of the choice and have a slight bias to selling after the up trend event as compared to selling after a TOD Event. Hence in summary, there is no evidence that the TOD Event contests the tenets the Efficient Market Hypothesis. 41 

# Chapter 5 - Research Conclusions & Recommendations 

## 5.1 Research Conclusions 

Since its introduction from Japan into the Western markets by Nison (1991) and Morris (1992), candlestick analysis has been widely employed by investment practitioners in a range of financial markets. More than 50 candlestick patterns have been studied within the academic community in effort to uncover the efficacy of their predictive power. Across the temporal spectrum and across the spectrum of candlestick patterns and markets, the results in the literature have been mixed. Some studies have reported moderate statistically significant evidence of predictive power for the majority of their respective patterns, suggesting that the Efficient Market Hypothesis and the theory or random walk may have been undermined (Caginalp & Laurent, 1998; Lu, Chen, & Hsu, 2015; Lu et al., 2012). However, other studies find that only a minority of their respective patterns have statistically significant evidence of predictive power, thus suggesting that the Efficient Market Hypothesis and theory of random walk remain mainly intact (Do Prado et al., 2013; Duvinage et al., 2013; Fock et al., 2005; Horton, 2009; Lu & Shiu, 2012; Lu, 2013, 2014; Marshall, Young, & Cahan, 2007; Marshall et al., 2006; Shiu & Lu, 2011). Within the literature, there are no academic studies that explicitly investigate small capitalization stocks in the United States and only two studies have considered the predictive power of the TOU and TOD Events for a time period that extends past 2005. As such, this study sought to fill this gap in the literature by examining the predictive power of the TOU Event and TOD Event for the period 1 June 2005 to 15 May 2015 for a sample of small capitalization stocks drawn from the S&P 600; and to determine if the findings undermine or support the Efficient Market Hypothesis. To enable the investigation into the predictive power of the TOU Event and TOD Event, the analysis herein made use of the normal approximation to binomial method based on Caginalp and Laurent (1998) to analyze 10 years of price data drawn from Reuters DataStream , for 47 small capitalization stocks drawn from the S&P 600, for three different dynamic holding periods. In consideration of the TOU Event, the results find no evidence to support the notion of predictive power for any of the three holding periods. To the contrary, all the Z-scores are negative, suggesting that the TOU Event has more success in predicting that prices will continue to go down, rather than reverse and go up. At a weak level of statistical significance, the Z-scores for Holding Period 2 and 3 are low enough to reject the opposite null hypothesis to that initially considered (that the TOU Event does not yield predictive power for prices to go down). 42 Similarly, the TOD Event analysis does not provide any level of statistical support to the proposition that the TOD Event has predictive power for the sample of stocks considered. For each holding period, the expected values (from the binomial model) for the number of successes ( nP0 ) vary only marginally from the actual success values (np ) derived from the sample. These results imply that there is limited difference between the price movements that follow a TOD Event compared to the price movements that follow an up trend. As with the TOU Event, considering transaction costs further degrades the case for the predictive power of the TOD Event. The results of the analysis find no evidence to reject the null hypothesis that the proportion of TOU Event or TOD Event successes for the sample of 47 small capitalization stocks drawn from the S&P 600 for the period 1 June 2005 to 15 May 2015, is less than or equal to the proportion of base case successes at a 95% confidence level. To the contrary, in the case of the TOU Event there is weak statistical evidence that the pattern accomplishes the opposite of what it is supposed to do and has some predictive power for prices to go down. In the case of the TOD Event, there is only a negligible difference between the price changes that occur after its manifestation as compared to base case up trends. While the findings cannot be generalized beyond the sample and dates investigated, they do imply that for the period 1 June 2005 to 15 May 2015 the TOU Event and TOD Event do not have predictive power for the 47 stocks sampled from the S&P 600. Hence, the findings imply that at a moderate level of statistical significance, there is no evidence to challenge the Efficient Market Hypothesis. 

## 5.2 Recommendations for Future Research 

This research could be further expanded by including the following aspects: 

 Incorporating a comparison of dynamic versus static holding returns so as to further explore the efficacy of candlestick analysis using dynamic holding period strategies. 

 Using clusters of small capitalization stocks that differ across geographical regions and sectors to determine whether candlestick analysis is sensitive to differences in market geography and firm capitalization size, as posited by Lu (2014). 

 Comparing the efficacy of candlestick analysis between developed and emerging markets to determine what differences exist in the applicability of candlestick analysis between markets. 43 

# References 

Anderson, D., Sweeney, D., & William, T. (1993). Statistics for business and economics (5th ed.). New York, NY: West Publishing Company. Anson, M. J. P., Chambers, D. R., Black, K. H., & Kazemi, H. (2012). CAIA level I: An introduction to core topics in alternative investments (2nd ed.). Hoboken, NJ: Wiley. Billingsley, R. S., & Chance, D. M. (1996). Benefits and limitations of diversification among commodity trading ddvisors. Journal of Portfolio Management , 23 (1), 65 –80. Block, S. B. (1999). Study of financial analysts: Practice and theory. Financial Analysts Journal , 55 (4), 86 –

95. Brennan, M. J., Chordia, T., & Subrahmanyam, A. (1998). Alternative factor specifications, security characteristics, and the cross-section of expected stock returns. Journal of Finacnial Economics ,

49 (3), 345 –373. Buchanan, D., & Huczynski, A. (2004). Organizational behaviour. An introductory text. New York, NY: Prentice Hall. Caginalp, G., & Balenovich, D. (2003). A theoretical foundation for technical analysis. Journal of Technical Analysis , 59 , 5 –21. Caginalp, G., & Laurent, H. (1998). The predictive power of price patterns. Applied Mathematical Finance ,

5, 181 –205. Chiang, Y., Ke, M., & Liang, T. (2012). Are technical trading strategies still profitable? Evidence from the Taiwan Stock Index Futures Market. Applied Financial Economics , 22 , 955 –965. Cross, F. (1973). The behavior of stock prices on Fridays and on Mondays. Financial Analysts Journal ,

29 (6), 67 –70. DeBondt, W. F. M., & Thaler, R. H. (1985). Does the stock market overreact? Journal of Finance , 40 , 793 –

808. DeBondt, W. F. M., & Thaler, R. H. (1987). Further evidence on investor overreaction and stock market seasonality. Journal of Finance , 42 , 557 –581. Do Prado, H. A., Ferneda, E., Morais, L. C. R., Luiz, A. J. B., & Matsura, E. (2013). On the effectiveness of candlestick chart analysis for the brazilian stock market. Procedia Computer Science , 22 , 1136 –1145. doi:10.1016/j.procs.2013.09.200 Duvinage, M., Mazza, P., & Petitjean, M. (2013). The intra-day performance of market timing strategies and trading systems based on Japanese candlesticks. Quantitative Finance , 13 (7), 1059 –1070. doi:10.1080/14697688.2013.768774 44 

Fama, E. (1965). Random walks in stock market prices. Financial Analysts Journal , 21 (5), 55 –59. Fama, E. (1970). Efficient capital markets: A review of theory and empirical work. The Journal of Financial Research , 25 (2), 383 –417. Fama, E., & French, K. R. (1992). The cross-section of expected stock returns. Journal of Finance , 47 (2), 427 –465. Fock, J. H., Klein, C., & Zwergel, B. (2005). Performance of candlestick analysis on intraday futures data. 

Journal of Derivatives , 13 (1), 28 –40. Haugen, R. A., & Baker, N. L. (1996). Commonality in the determinants of expected stock returns. Journal of Finacnial Economics , 41 (3), 401 –439. Horton, M. J. (2009). Stars, crows, and doji: The use of candlesticks in stock selection. The Quarterly Review of Economics and Finance , 49 (2), 283 –294. doi:10.1016/j.qref.2007.10.005 Hossein, V., & Kalash, S. (2008). Testing market efficiency for different market capitalization funds. 

American Journal of Business , 23 (2), 17 –28. Hung, J.-C., Lee, Y.-H., & Pai, T.-Y. (2009). Examining market efficiency for large- and small-capitalization of TOPIX and FTSE stock indices. Applied Financial Economics , 19 (9), 735 –744. doi:10.1080/09603100802129775 Hyde, K. F. (2000). Recognising deductive processes in qualitative research. Qualitative Market Research: An International Journal , 3(2), 82 –90. doi:10.1108/13522750010322089 Keller, G. (2009). Managerial statistics (8th ed.). Mason, OH: Cengage. Kilgallen, T. (2012). Testing the simple moving average across commodities, global stock indices, and currencies. The Journal of Wealth Management , 15 (1), 82 –101. Lakonishok, J., & Smidt, S. (1988). Are seasonal anomolies real? A ninety-year perspective. Review of Financial Studies , 1(4), 403 –425. Lo, A. W., & Repin, D. V. (2002). The psychophysiology of real time financial risk processing. Journal of Cognitive Neuroscience , I4 (3), 323 –39. Lo, A. W., Repin, D. V., & Steenbarger, B. N. (2005). Fear and greed in financial markets: A clinical study of day-traders. Cognitive Neuroscientific Foundations of Behavior , 95 (2), 352 –359. Lu, T.-H. (2013). Candlestick charting in European stock markets. The Finsia Journal of Applied Finance ,(2). Lu, T.-H. (2014). The profitability of candlestick charting in the Taiwan stock market. Pacific-Basin Finance Journal , 26 , 65 –78. doi:10.1016/j.pacfin.2013.10.006 45 

Lu, T.-H., Chen, Y.-C., & Hsu, Y.-C. (2015). Trend definition or holding strategy: What determines the profitability of candlestick charting? Journal of Banking & Finance , 61 , 172 –183. doi:10.1016/j.jbankfin.2015.09.009 Lu, T.-H., & Shiu, Y.-M. (2012). Tests for two-day candlestick patterns in the emerging equity market of Taiwan. Emerging Markets Finance and Trade , 48 (1), 41 –57. doi:10.2753/REE1540-496X4801S104 Lu, T.-H., Shiu, Y.-M., & Liu, T.-C. (2012). Profitable candlestick trading strategies —The evidence from a new perspective. Review of Financial Economics , 21 (2), 63 –68. doi:10.1016/j.rfe.2012.02.001 Malkiel, B. G. (1995). Returns from investing in equity mutual funds 1971 to 1991. The Journal of Finance ,

1(2), 549 –573. Malkiel, B. G. (2003). The efficient market hypothesis and its critics. Journal of Economic Perspectives ,

17 (1), 59 –82. Marshall, B. R., Young, M. R., & Cahan, R. (2007). Are candlestick technical trading strategies profitable in the Japanese equity market? Review of Quantitative Finance and Accounting , 31 (2), 191 –207. doi:10.1007/s11156-007-0068-1 Marshall, B. R., Young, M. R., & Rose, L. C. (2006). Candlestick technical trading strategies: Can they create value for investors? Journal of Banking & Finance , 30 (8), 2303 –2323. doi:10.1016/j.jbankfin.2005.08.001 Menkhoff, L. (1997). Examining the use of technical currency analysis. International Journal of Finance and Economics , 2(4), 307 – 318. Morris, G. L. (1992). Candlepower, advanced candlestick pattern recognition and filtering techniques for trading stocks and futures . Chicago, IL: Probus. Morris, G. L. (2006). Candlestick charting explained: Timeless techniques for trading and futures (3rd ed.). New York, NY: McGraw-Hill. Nison, S. (1991). Japanese candlestick charting techniques: A contemporary guide to the ancient investment techniques of the far east . New York, NY: New York Institute of Finance. Panagiotidis, T. (2005). Market capitalization and efficiency. Does it matter? Evidence from the Athens Stock Exchange. Applied Financial Economics , 15 (10), 707 –713. doi:10.1080/09603100500107883 Park, C.-H., & Irwin, S. H. (2007). What do we know about the profitability of technical analysis? Journal of Economic Surveys , 21 (4), 786 –826. doi:10.1111/j.1467-6419.2007.00519.x Ricciardi, V., & Simon, H. K. (2000). What is behavioral finance? Business, Education and Technology Journal , 2(2), 1 –9. Rouwenhorst, K. G. (1999). Local return factors and turnover in emerging stock markets. The Journal of Finance , 54 (4), 439 –1464. 46 

Rozeff, M., & Kinny, W. (1976). Capital market seaonality: The case of stock returns. Journal of Finacnial Economics , 3(4), 379 –402. S&P Dow Jones. (2014, October 31). S&P smallcap 600. S&P Dow Jones Indices . Retrieved from http://www.spdji.com Shiu, Y.-M., & Lu, T.-H. (2011). Pinpoint and synergistic trading strategies of candlesticks. International Journal of Economics and Finance , 3(1), 234 –244. Smitten, R., & Livermore, J. (2001). How to trade in stocks: His own words. The Jesse Livermore secret trading formula for understanding timing, money management and emotional control . New York, NY: McGraw-Hill. Statman, M. (1999). Behaviorial finance: Past battles and future engagements. Financial Analysts Journal , 55 (6), 18 –27. Steenbarger, B. N. (2002). The psychology of trading: Tools and techniques for minding the markets . New York, NY: Wiley. Stracca, L. (2004). Behavioral finance and asset prices: Where do we stand? Journal of Economic Psychology , 25 (3), 373 –405. doi:10.1016/S0167-4870(03)00055-2 Sullivan, R., Timmermann, A., & White, H. (1999). Data-snooping, technical trading rule performance, and the bootstrap. The Journal of Finance , 54 (5), 1647 –1691. Thaler, R. H. (1994). The winners curse: Paradoxes and anomalies of economic life . New York, NY: Free Press. The Institute of Chartered Financial Analysts. (2012). CFA derivatives and portfolio management . (Level 2 Volume 6) Charlottesville, VA: Wiley. 47 

# Appendix I 

Figure 5: Sample List from the S&P 600 2005 2015 

Name Symbol 

Market 

Capitalization Name Symbol 

Market 

Capitalization 

Allete Inc ALE 1,424,496,000 $ Allete Inc ALE 2,471,188,190 $Anixter Intl Inc AXE 1,418,517,770 $ Anixter Intl Inc AXE 2,139,109,050 $Benchmark Electronics Inc BHE 1,308,604,400 $ Benchmark Electronics Inc BHE 1,166,745,500 $Brady Corporation BRC 1,498,296,840 $ Brady Corporation BRC 1,207,489,010 $Briggs & Stratton Corp BGG 1,744,907,380 $ Briggs & Stratton Corp BGG 816,626,000 $Carbo Ceramics Inc CRR 1,146,813,120 $ Carbo Ceramics Inc CRR 773,794,000 $Chemed Corp CHE 1,054,378,640 $ Chemed Corp CHE 2,594,167,070 $Coherent Inc COHR 1,004,368,190 $ Coherent Inc COHR 1,408,173,790 $Curtiss-Wright Corp CW 1,171,464,320 $ Curtiss-Wright Corp CW 3,240,397,700 $EPR Properties EPR 1,114,636,000 $ EPR Properties EPR 3,176,110,440 $Ethan Allen Interiors ETH 1,107,537,600 $ Ethan Allen Interiors ETH 899,155,940 $First Bancorp (Puerto Rico) FBP 1,545,197,290 $ First Bancorp (Puerto Rico) FBP 876,625,100 $First Midwest Bancorp (IL) FMBI 1,586,894,050 $ First Midwest Bancorp (IL) FMBI 1,475,896,380 $Haemonetics Corp HAE 1,053,560,200 $ Haemonetics Corp HAE 2,046,129,970 $Headwaters Inc HW 1,322,803,040 $ Headwaters Inc HW 1,552,353,300 $Heartland Express Inc HTLD 1,503,750,000 $ Heartland Express Inc HTLD 1,873,459,940 $Invacare Corp IVC 1,402,315,350 $ Invacare Corp IVC 556,065,060 $Knight Transportation Inc KNX 1,381,791,750 $ Knight Transportation Inc KNX 2,223,124,540 $Lexington Realty Trust LXP 1,116,241,070 $ Lexington Realty Trust LXP 2,064,483,750 $Men's Wearhouse Inc MW 1,857,230,980 $ Men's Wearhouse Inc MW 2,754,297,000 $Meritage Corp MTH 1,870,358,400 $ Meritage Corp MTH 1,711,454,400 $Microsemi Corp MSCC 1,260,245,440 $ Microsemi Corp MSCC 3,218,852,320 $Moog Inc A MOG.A 1,171,585,875 $ Moog Inc A MOG.A 2,547,454,590 $New Jersey Resources Corp NJR 1,200,787,500 $ New Jersey Resources Corp NJR 2,438,202,690 $Piedmont Natural Gas Inc PNY 1,874,247,500 $ Piedmont Natural Gas Inc PNY 3,038,937,650 $ProAssurance Corp PRA 1,141,834,560 $ ProAssurance Corp PRA 2,673,680,400 $Progress Software Corp PRGS 1,066,017,650 $ Progress Software Corp PRGS 1,501,183,320 $Quanex Building Products Corp NX 1,293,669,590 $ Quanex Building Products Corp NX 675,726,450 $RLI Corp RLI 1,110,952,260 $ RLI Corp RLI 2,407,410,600 $Pool Corp POOL 1,874,747,160 $ Pool Corp POOL 3,083,273,640 $SEACOR Holdings Inc. CKH 1,062,149,640 $ SEACOR Holdings Inc. CKH 1,186,185,000 $Selective Insurance Group Inc SIGI 1,346,147,970 $ Selective Insurance Group Inc SIGI 1,783,611,360 $Simpson Manufacturing Co Inc SSD 1,379,175,120 $ Simpson Manufacturing Co Inc SSD 1,782,389,600 $SkyWest Inc SKYW 1,051,882,560 $ SkyWest Inc SKYW 893,219,660 $Sonic Corp SONC 2,045,780,000 $ Sonic Corp SONC 1,506,440,000 $Standard Pacific Corp SPF 2,709,017,440 $ Standard Pacific Corp SPF 2,392,982,680 $Stein Mart Inc SMRT 1,023,416,160 $ Stein Mart Inc SMRT 463,029,000 $Stone Energy Corp SGY 1,148,126,460 $ Stone Energy Corp SGY 348,859,000 $Take-Two Interactive Software TTWO 1,780,956,240 $ Take-Two Interactive Software TTWO 2,617,802,460 $Children's Place, Inc PLCE 1,260,936,450 $ Children's Place, Inc PLCE 1,218,258,530 $Toro Co TTC 1,861,223,700 $ Toro Co TTC 3,820,187,580 $Unit Corp UNT 1,784,696,760 $ Unit Corp UNT 875,969,380 $United Bankshares Inc (WV) UBSI 1,442,598,150 $ United Bankshares Inc (WV) UBSI 2,810,560,560 $Watts Industries Inc A WTS 1,126,854,990 $ Watts Industries Inc A WTS 1,985,092,170 $Winnebago Industries Inc WGO 1,101,031,890 $ Winnebago Industries Inc WGO 599,753,370 $Wintrust Financial Corp WTFC 1,175,192,970 $ Wintrust Financial Corp WTFC 2,567,793,000 $Wolverine World Wide Inc WWW 1,318,017,900 $ Wolverine World Wide Inc WWW 3,019,126,500 $


