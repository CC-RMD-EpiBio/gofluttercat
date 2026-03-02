#!/usr/bin/env python3
"""Extract IRT item parameters from grm_imputed params.h5 and generate
item JSON files for gofluttercat's embedded instrument packages.

Usage:
    python extract_items.py

Reads from bayesianquilts/notebooks/irt/{instrument}/grm_imputed/params.h5
and writes JSON files to gofluttercat/backend-golang/{instrument}/factorized/
"""

import json
import os
import sys

import h5py
import numpy as np

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
BQ_IRT = os.path.join(os.path.dirname(REPO_ROOT), "bayesianquilts", "notebooks", "irt")
BACKEND = os.path.join(REPO_ROOT, "backend-golang")


def softplus(x):
    return np.log1p(np.exp(x))


def extract_params(h5_path):
    """Extract discrimination and threshold parameters from grm_imputed params.h5."""
    with h5py.File(h5_path, "r") as f:
        disc_loc = np.squeeze(f["params/discriminations\\softplus\\normal\\loc"][:])
        diff0_loc = np.squeeze(f["params/difficulties0\\identity\\normal\\loc"][:])

        has_ddiff = "params/ddifficulties\\softplus\\normal\\loc" in f
        if has_ddiff:
            ddiff_loc = np.squeeze(f["params/ddifficulties\\softplus\\normal\\loc"][:])
        else:
            ddiff_loc = None

    disc = softplus(disc_loc)  # shape: (n_items,)
    diff0 = diff0_loc  # shape: (n_items,)

    n_items = disc.shape[0]
    thresholds = []
    for i in range(n_items):
        if ddiff_loc is not None and ddiff_loc.ndim == 2:
            ddiff_vals = softplus(ddiff_loc[i])
            t = np.cumsum(np.concatenate([[diff0[i]], ddiff_vals]))
        else:
            t = np.array([diff0[i]])
        thresholds.append(t.tolist())

    return disc.tolist(), thresholds


# ----- Instrument definitions -----

GRIT_ITEMS = {
    "GS1": "I have overcome setbacks to conquer an important challenge.",
    "GS2": "New ideas and projects sometimes distract me from previous ones.",
    "GS3": "My interests change from year to year.",
    "GS4": "Setbacks don't discourage me.",
    "GS5": "I have been obsessed with a certain idea or project for a short time but later lost interest.",
    "GS6": "I am a hard worker.",
    "GS7": "I often set a goal but later choose to pursue a different one.",
    "GS8": "I have difficulty maintaining my focus on projects that take more than a few months to complete.",
    "GS9": "I finish whatever I begin.",
    "GS10": "I have achieved a goal that took years of work.",
    "GS11": "I become interested in new pursuits every few months.",
    "GS12": "I am diligent.",
}

GRIT_RESPONSES = {
    "0": {"text": "very much like me", "value": 0},
    "1": {"text": "mostly like me", "value": 1},
    "2": {"text": "somewhat like me", "value": 2},
    "3": {"text": "not much like me", "value": 3},
    "4": {"text": "not like me at all", "value": 4},
}

NPI_ITEMS = {
    "Q1": ("I have a natural talent for influencing people.", "I am not good at influencing people."),
    "Q2": ("Modesty doesn't become me.", "I am essentially a modest person."),
    "Q3": ("I would do almost anything on a dare.", "I tend to be a fairly cautious person."),
    "Q4": ("When people compliment me I sometimes get embarrassed.", "I know that I am good because everybody keeps telling me so."),
    "Q5": ("The thought of ruling the world frightens the hell out of me.", "If I ruled the world it would be a better place."),
    "Q6": ("I can usually talk my way out of anything.", "I try to accept the consequences of my behavior."),
    "Q7": ("I prefer to blend in with the crowd.", "I like to be the center of attention."),
    "Q8": ("I will be a success.", "I am not too concerned about success."),
    "Q9": ("I am no better or worse than most people.", "I think I am a special person."),
    "Q10": ("I am not sure if I would make a good leader.", "I see myself as a good leader."),
    "Q11": ("I am assertive.", "I wish I were more assertive."),
    "Q12": ("I like to have authority over other people.", "I don't mind following orders."),
    "Q13": ("I find it easy to manipulate people.", "I don't like it when I find myself manipulating people."),
    "Q14": ("I insist upon getting the respect that is due me.", "I usually get the respect that I deserve."),
    "Q15": ("I don't particularly like to show off my body.", "I like to show off my body."),
    "Q16": ("I can read people like a book.", "People are sometimes hard to understand."),
    "Q17": ("If I feel competent I am willing to take responsibility for making decisions.", "I like to take responsibility for making decisions."),
    "Q18": ("I just want to be reasonably happy.", "I want to amount to something in the eyes of the world."),
    "Q19": ("My body is nothing special.", "I like to look at my body."),
    "Q20": ("I try not to be a show off.", "I will usually show off if I get the chance."),
    "Q21": ("I always know what I am doing.", "Sometimes I am not sure of what I am doing."),
    "Q22": ("I sometimes depend on people to get things done.", "I rarely depend on anyone else to get things done."),
    "Q23": ("Sometimes I tell good stories.", "Everybody likes to hear my stories."),
    "Q24": ("I expect a great deal from other people.", "I like to do things for other people."),
    "Q25": ("I will never be satisfied until I get all that I deserve.", "I take my satisfactions as they come."),
    "Q26": ("Compliments embarrass me.", "I like to be complimented."),
    "Q27": ("I have a strong will to power.", "Power for its own sake doesn't interest me."),
    "Q28": ("I don't care about new fads and fashions.", "I like to start new fads and fashions."),
    "Q29": ("I like to look at myself in the mirror.", "I am not particularly interested in looking at myself in the mirror."),
    "Q30": ("I really like to be the center of attention.", "It makes me uncomfortable to be the center of attention."),
    "Q31": ("I can live my life in any way I want to.", "People can't always live their lives in terms of what they want."),
    "Q32": ("Being an authority doesn't mean that much to me.", "People always seem to recognize my authority."),
    "Q33": ("I would prefer to be a leader.", "It makes little difference to me whether I am a leader or not."),
    "Q34": ("I am going to be a great person.", "I hope I am going to be successful."),
    "Q35": ("People sometimes believe what I tell them.", "I can make anybody believe anything I want them to."),
    "Q36": ("I am a born leader.", "Leadership is a quality that takes a long time to develop."),
    "Q37": ("I wish somebody would someday write my biography.", "I don't like people to pry into my life for any reason."),
    "Q38": ("I get upset when people don't notice how I look when I go out in public.", "I don't mind blending into the crowd when I go out in public."),
    "Q39": ("I am more capable than other people.", "There is a lot that I can learn from other people."),
    "Q40": ("I am much like everybody else.", "I am an extraordinary person."),
}

# NPI scoring key: which choice (1 or 2) is the narcissistic response.
# score += (choice == narcissistic_choice)
NPI_NARCISSISTIC_CHOICE = {
    "Q1": 1, "Q2": 1, "Q3": 1, "Q4": 2, "Q5": 2, "Q6": 1, "Q7": 2, "Q8": 1,
    "Q9": 2, "Q10": 2, "Q11": 1, "Q12": 1, "Q13": 1, "Q14": 1, "Q15": 2, "Q16": 1,
    "Q17": 2, "Q18": 2, "Q19": 2, "Q20": 2, "Q21": 1, "Q22": 2, "Q23": 2, "Q24": 1,
    "Q25": 1, "Q26": 2, "Q27": 1, "Q28": 2, "Q29": 1, "Q30": 1, "Q31": 1, "Q32": 2,
    "Q33": 1, "Q34": 1, "Q35": 2, "Q36": 1, "Q37": 1, "Q38": 1, "Q39": 1, "Q40": 2,
}

TMA_ITEMS = {
    "Q1": "I do not tire quickly.",
    "Q2": "I am troubled by attacks of nausea.",
    "Q3": "I believe I am no more nervous than most others.",
    "Q4": "I have very few headaches.",
    "Q5": "I work under a great deal of tension.",
    "Q6": "I cannot keep my mind on one thing.",
    "Q7": "I worry over money and business.",
    "Q8": "I frequently notice my hand shakes when I try to do something.",
    "Q9": "I blush no more often than others.",
    "Q10": "I have diarrhea once a month or more.",
    "Q11": "I worry quite a bit over possible misfortunes.",
    "Q12": "I practically never blush.",
    "Q13": "I am often afraid that I am going to blush.",
    "Q14": "I have nightmares every few nights.",
    "Q15": "My hands and feet are usually warm.",
    "Q16": "I sweat very easily even on cool days.",
    "Q17": "Sometimes when embarrassed, I break out in a sweat.",
    "Q18": "I hardly ever notice my heart pounding and I am seldom short of breath.",
    "Q19": "I feel hungry almost all the time.",
    "Q20": "I am very seldom troubled by constipation.",
    "Q21": "I have a great deal of stomach trouble.",
    "Q22": "I have had periods in which I lost sleep over worry.",
    "Q23": "My sleep is fitful and disturbed.",
    "Q24": "I dream frequently about things that are best kept to myself.",
    "Q25": "I am easily embarrassed.",
    "Q26": "I am more sensitive than most other people.",
    "Q27": "I frequently find myself worrying about something.",
    "Q28": "I wish I could be as happy as others seem to be.",
    "Q29": "I am usually calm and not easily upset.",
    "Q30": "I cry easily.",
    "Q31": "I feel anxiety about something or someone almost all the time.",
    "Q32": "I am happy most of the time.",
    "Q33": "It makes me nervous to have to wait.",
    "Q34": "I have periods of such great restlessness that I cannot sit long in a chair.",
    "Q35": "Sometimes I become so excited that I find it hard to get to sleep.",
    "Q36": "I have sometimes felt that difficulties were piling up so high that I could not overcome them.",
    "Q37": "I must admit that I have at times been worried beyond reason over something that really did not matter.",
    "Q38": "I have very few fears compared to my friends.",
    "Q39": "I have been afraid of things or people that I know could not hurt me.",
    "Q40": "I certainly feel useless at times.",
    "Q41": "I find it hard to keep my mind on a task or job.",
    "Q42": "I am usually self-conscious.",
    "Q43": "I am inclined to take things hard.",
    "Q44": "I am a high-strung person.",
    "Q45": "Life is a trial for me much of the time.",
    "Q46": "At times I think I am no good at all.",
    "Q47": "I am certainly lacking in self-confidence.",
    "Q48": "I sometimes feel that I am about to go to pieces.",
    "Q49": "I shrink from facing crisis or difficulty.",
    "Q50": "I am entirely self-confident.",
}

WPI_ITEMS = {
    "Q1": "Do you usually feel well and strong?",
    "Q2": "Do you usually sleep well?",
    "Q3": "Are you often frightened in the middle of the night?",
    "Q4": "Are you troubled with dreams about your work?",
    "Q5": "Do you have nightmares?",
    "Q6": "Do you have too many sexual dreams?",
    "Q7": "Do you ever walk in your sleep?",
    "Q8": "Do you have the sensation of falling when going to sleep?",
    "Q9": "Does your heart ever thump in your ears so that you cannot sleep?",
    "Q10": "Do ideas run through your head so that you cannot sleep?",
    "Q11": "Do you feel well rested in the morning?",
    "Q12": "Do your eyes often pain you?",
    "Q13": "Do things ever seem to swim or get misty before your eyes?",
    "Q14": "Do you often have the feeling of suffocating?",
    "Q15": "Do you have continual itchings in the face?",
    "Q16": "Are you bothered much by blushing?",
    "Q17": "Are you bothered by fluttering of the heart?",
    "Q18": "Do you feel tired most of the time?",
    "Q19": "Have you ever had fits of dizziness?",
    "Q20": "Do you have queer, unpleasant feelings in any part of the body?",
    "Q21": "Do you ever feel an awful pressure in or about the head?",
    "Q22": "Do you often have bad pains in any part of the body?",
    "Q23": "Do you have a great many bad headaches?",
    "Q24": "Is your head apt to ache on one side?",
    "Q25": "Have you ever fainted away?",
    "Q26": "Have you often fainted away?",
    "Q27": "Have you ever been blind, half-blind, deaf or dumb for a time?",
    "Q28": "Have you ever had an arm or leg paralyzed?",
    "Q29": "Have you ever lost your memory for a time?",
    "Q30": "Did you have a happy childhood?",
    "Q31": "Were you happy when 14 to 18 years old?",
    "Q32": "Were you considered a bad boy/girl?",
    "Q33": "As a child did you like to play alone better than to play with other children?",
    "Q34": "Did the other children let you play with them?",
    "Q35": "Were you shy with other boys/girls?",
    "Q36": "Did you ever run away from home?",
    "Q37": "Did you ever have a strong desire to run away from home?",
    "Q38": "Has your family always treated you right?",
    "Q39": "Did the teachers in school generally treat you right?",
    "Q40": "Have your employers generally treated you right?",
    "Q41": "Do you know of anybody who is trying to do you harm?",
    "Q42": "Do people find fault with you more than you deserve?",
    "Q43": "Do you make friends easily?",
    "Q44": "Did you ever make love to a girl/boy?",
    "Q45": "Do you get used to new places quickly?",
    "Q46": "Do you find your way about easily?",
    "Q47": "Does liquor make you quarrelsome?",
    "Q48": "Do you think drinking has hurt you?",
    "Q49": "Do you think tobacco has hurt you?",
    "Q50": "Do you think you have hurt yourself by going too much with women/men?",
    "Q51": "Have you hurt yourself by masturbation?",
    "Q52": "Did you ever think you had lost your manhood/womanhood?",
    "Q53": "Have you ever had any great mental shock?",
    "Q54": "Have you ever seen a vision?",
    "Q55": "Did you ever have the habit of taking any form of dope?",
    "Q56": "Do you have trouble in walking in the dark?",
    "Q57": "Have you ever felt as if someone was hypnotizing you and making you act against your will?",
    "Q58": "Are you ever bothered by the feeling that people are reading your thoughts?",
    "Q59": "Do you ever have a queer feeling as if you were not your old self?",
    "Q60": "Are you ever bothered by a feeling that things are not real?",
    "Q61": "Are you troubled with the idea that people are watching you on the street?",
    "Q62": "Are you troubled with the fear of being crushed in a crowd?",
    "Q63": "Does it make you uneasy to cross a bridge over a river?",
    "Q64": "Does it make you uneasy to go into a tunnel?",
    "Q65": "Does it make you uneasy to have to cross a wide street or open square?",
    "Q66": "Does it make you uneasy to sit in a small room with the door shut?",
    "Q67": "Do you usually know just what you want to do next?",
    "Q68": "Do you worry too much about little things?",
    "Q69": "Do you think you worry too much when you have an unfinished job on your hands?",
    "Q70": "Do you think you have too much trouble in making up your mind?",
    "Q71": "Can you do good work while people are looking on?",
    "Q72": "Do you get rattled easily?",
    "Q73": "Can you sit still without fidgeting?",
    "Q74": "Does your mind wander badly so that you lose track of what you are doing?",
    "Q75": "Does some particular useless thought keep coming into your mind to bother you?",
    "Q76": "Can you do the little chores of the day without worrying over them?",
    "Q77": "Do you feel you must do a thing over several times before you can drop it?",
    "Q78": "Are you afraid of responsibility?",
    "Q79": "Do you feel like jumping off when you are on a high place?",
    "Q80": "Are you troubled at night with the idea that somebody is following you?",
    "Q81": "Do you find it difficult to pass urine in the presence of others?",
    "Q82": "Do you have a great fear of fire?",
    "Q83": "Do you ever feel a strong desire to go and set fire to something?",
    "Q84": "Do you ever feel a strong desire to go steal things?",
    "Q85": "Did you ever have the habit of biting your finger nails?",
    "Q86": "Did you ever have the habit of stuttering?",
    "Q87": "Did you ever have the habit of twitching your face, neck or shoulders?",
    "Q88": "Did you ever have the habit of wetting the bed?",
    "Q89": "Are you troubled with shyness?",
    "Q90": "Have you a good appetite?",
    "Q91": "Is it easy to make you laugh?",
    "Q92": "Is it easy to get you angry?",
    "Q93": "Is it easy to get you cross or grouchy?",
    "Q94": "Do you get tired of people quickly?",
    "Q95": "Do you get tired of amusements quickly?",
    "Q96": "Do you get tired of work?",
    "Q97": "Do your interests change frequently?",
    "Q98": "Do your feelings keep changing from happy to sad and from sad to happy without any reason?",
    "Q99": "Do you feel sad or low-spirited most of the time?",
    "Q100": "Did you ever have a strong desire to commit suicide?",
    "Q101": "Did you ever have St Vitus' dance?",
    "Q102": "Did you ever have convulsions?",
    "Q103": "Did you ever have heart disease?",
    "Q104": "Did you ever have anemia badly?",
    "Q105": "Did you ever have dyspepsia (indigestion)?",
    "Q106": "Did you ever have asthma or hay fever (allergies)?",
    "Q107": "Did you ever have a nervous breakdown?",
    "Q108": "Have you ever been afraid of going insane?",
    "Q109": "Has any of your family been insane, epileptic, or feebleminded?",
    "Q110": "Has any of your family committed suicide?",
    "Q111": "Has any of your family had a drug habit?",
    "Q112": "Has any of your family been a drunkard?",
    "Q113": "Can you stand the sight of blood?",
    "Q114": "Can you stand pain quietly?",
    "Q115": "Can you stand disgusting smells?",
    "Q116": "Do you like outdoor life?",
}


def build_grit_items(discs, thresholds):
    """Build GRIT item JSON dicts."""
    item_names = [f"GS{i}" for i in range(1, 13)]
    items = []
    for idx, name in enumerate(item_names):
        item = {
            "item": name,
            "question": GRIT_ITEMS[name],
            "responses": dict(GRIT_RESPONSES),
            "scales": {
                "grit": {
                    "discrimination": discs[idx],
                    "difficulties": thresholds[idx],
                }
            },
        }
        items.append((name, item))
    return items


def build_npi_items(discs, thresholds):
    """Build NPI item JSON dicts."""
    item_names = [f"Q{i}" for i in range(1, 41)]
    items = []
    for idx, name in enumerate(item_names):
        choice_a, choice_b = NPI_ITEMS[name]
        narc = NPI_NARCISSISTIC_CHOICE[name]
        if narc == 1:
            responses = {
                "1": {"text": choice_a, "value": 1},
                "0": {"text": choice_b, "value": 0},
            }
        else:
            responses = {
                "1": {"text": choice_b, "value": 1},
                "0": {"text": choice_a, "value": 0},
            }
        item = {
            "item": name,
            "question": "Which statement fits you better?",
            "responses": responses,
            "scales": {
                "narcissism": {
                    "discrimination": discs[idx],
                    "difficulties": thresholds[idx],
                }
            },
        }
        items.append((name, item))
    return items


def build_tma_items(discs, thresholds):
    """Build TMA item JSON dicts."""
    item_names = [f"Q{i}" for i in range(1, 51)]
    items = []
    for idx, name in enumerate(item_names):
        item = {
            "item": name,
            "question": TMA_ITEMS[name],
            "responses": {
                "1": {"text": "true", "value": 1},
                "0": {"text": "false", "value": 0},
            },
            "scales": {
                "anxiety": {
                    "discrimination": discs[idx],
                    "difficulties": thresholds[idx],
                }
            },
        }
        items.append((name, item))
    return items


def build_wpi_items(discs, thresholds):
    """Build WPI item JSON dicts."""
    item_names = [f"Q{i}" for i in range(1, 117)]
    items = []
    for idx, name in enumerate(item_names):
        item = {
            "item": name,
            "question": WPI_ITEMS[name],
            "responses": {
                "1": {"text": "yes", "value": 1},
                "0": {"text": "no", "value": 0},
            },
            "scales": {
                "psychoneurosis": {
                    "discrimination": discs[idx],
                    "difficulties": thresholds[idx],
                }
            },
        }
        items.append((name, item))
    return items


def build_rwa_items(discs, thresholds):
    """Build RWA item JSON dicts, reusing question text and responses
    from the existing factorized JSON files."""
    rwa_factorized = os.path.join(BACKEND, "rwas", "factorized")
    item_names = [f"Q{i}" for i in range(1, 23)]
    items = []
    for idx, name in enumerate(item_names):
        existing_path = os.path.join(rwa_factorized, f"{name}.json")
        with open(existing_path) as f:
            existing = json.load(f)
        item = {
            "item": name,
            "question": existing["question"],
            "responses": existing["responses"],
            "scales": {
                "rwa": {
                    "discrimination": discs[idx],
                    "difficulties": thresholds[idx],
                }
            },
        }
        items.append((name, item))
    return items


INSTRUMENTS = {
    "grit": {
        "builder": build_grit_items,
        "scale_name": "grit",
    },
    "npi": {
        "builder": build_npi_items,
        "scale_name": "narcissism",
    },
    "tma": {
        "builder": build_tma_items,
        "scale_name": "anxiety",
    },
    "wpi": {
        "builder": build_wpi_items,
        "scale_name": "psychoneurosis",
    },
    "rwa": {
        "builder": build_rwa_items,
        "scale_name": "rwa",
    },
}


def main():
    for instrument, cfg in INSTRUMENTS.items():
        h5_path = os.path.join(BQ_IRT, instrument, "grm_imputed", "params.h5")
        if not os.path.exists(h5_path):
            print(f"SKIP {instrument}: {h5_path} not found")
            continue

        discs, thresholds = extract_params(h5_path)
        items = cfg["builder"](discs, thresholds)

        out_dir = os.path.join(BACKEND, instrument, "factorized")
        os.makedirs(out_dir, exist_ok=True)

        for name, item_dict in items:
            out_path = os.path.join(out_dir, f"{name}.json")
            with open(out_path, "w") as f:
                json.dump(item_dict, f, indent=2)

        print(f"{instrument}: wrote {len(items)} items to {out_dir}")


if __name__ == "__main__":
    main()
